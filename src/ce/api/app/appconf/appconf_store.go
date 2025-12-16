package appconf

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"mime"
	"path"
	"strings"
	"text/template"

	"context"

	"github.com/dlclark/regexp2"
	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/authwall"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/deploy"
	"github.com/stormkit-io/stormkit-io/src/lib/config"

	"github.com/stormkit-io/stormkit-io/src/lib/database"
	"github.com/stormkit-io/stormkit-io/src/lib/slog"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

type statement struct {
	selectResetCacheArgs string
	selectConfigs        string
}

var stmt = &statement{
	selectResetCacheArgs: `
		SELECT
			COALESCE(d.domain_name, ''), a.display_name
		FROM apps_build_conf e
			LEFT JOIN domains d ON e.env_id = d.env_id AND d.domain_verified IS TRUE
			LEFT JOIN apps a ON a.app_id = e.app_id
		WHERE
			e.env_id = $1 AND
			a.display_name IS NOT NULL;
	`,

	selectConfigs: `
		WITH deployment AS (
			SELECT
				d.deployment_id,
				d.env_id,
				d.app_id,
				coalesce(d.upload_result->>'serverLocation', '')   	 as fn_loc,
				coalesce(d.upload_result->>'clientLocation', '')     as st_loc,
				coalesce(d.upload_result->>'serverlessLocation', '') as api_loc,
				coalesce(d.api_path_prefix, '/api')    	 			 as api_path_prefix,
				d.build_manifest 						 			 as manifest,
				e.env_name,
				e.updated_at							 			 as env_updated,
				e.build_conf							 			 as build_conf,
				e.auth_wall_conf						 			 as auth_wall_conf,
				coalesce(dp.percentage_released, 0)		 			 as percentage,
				a.display_name,
				coalesce(u.metadata->>'package', 'free') 			 as subscription_tier,
				u.user_id							 	 			 as billing_user_id,
				{{ or .columns "'' as cert_value, '' as cert_key, 0 as domain_id" }}
			FROM
				deployments d
			INNER JOIN
				apps_build_conf e ON d.env_id = e.env_id
			INNER JOIN
				apps a ON a.app_id = d.app_id
			INNER JOIN
				teams t ON t.team_id = a.team_id
			INNER JOIN
				users u ON u.user_id = t.user_id
			LEFT JOIN
				deployments_published dp ON d.deployment_id = dp.deployment_id
			{{ .join }}
			WHERE
				d.deleted_at IS NULL AND
				e.deleted_at IS NULL AND
				a.deleted_at IS NULL AND
				{{ .where }}
		),
		snippets AS (
			SELECT
				json_agg(
					distinct jsonb_build_object(
						'content', s.snippet_content,
						'location', s.snippet_location,
						'prepend', s.should_prepend,
						'rules', s.snippet_rules
					)
				) as json_data
			FROM snippets s
			WHERE
				s.snippet_id IS NOT NULL AND
				s.is_enabled IS TRUE AND
				s.env_id = (SELECT env_id FROM deployment LIMIT 1) AND
				(
					s.snippet_rules IS NULL OR 
					s.snippet_rules->'hosts' = '[]' OR
					s.snippet_rules->'hosts' IS NULL OR
					s.snippet_rules->'hosts' ? $1
				)
		)
		SELECT
			d.app_id, d.deployment_id, d.env_id,
			d.fn_loc, d.st_loc, d.api_loc, d.api_path_prefix,
			d.manifest, d.env_updated, d.build_conf, d.percentage,
			coalesce(d.cert_value, '') as cert_value,
			coalesce(d.cert_key, '') as cert_key,
			d.domain_id, d.auth_wall_conf,
			(SELECT json_data FROM snippets) as snippets,
			d.display_name, d.env_name, d.subscription_tier, d.billing_user_id
		FROM deployment d
	`,
}

// Store is the store to handle appconf logic
type Store struct {
	*database.Store
	selectConfigs *template.Template
}

// NewStore returns a store instance.
func NewStore() *Store {
	return &Store{
		Store:         database.NewStore(),
		selectConfigs: template.Must(template.New("query").Parse(stmt.selectConfigs)),
	}
}

type ConfigFilters struct {
	EnvName      string
	HostName     string
	DisplayName  string
	DeploymentID types.ID
}

func (s *Store) queryWithDeploymentID(filters ConfigFilters) (string, []any, error) {
	var wr bytes.Buffer

	err := s.selectConfigs.Execute(&wr, map[string]any{
		"where": "d.deployment_id = $2 AND a.display_name = $3",
		"join":  "",
	})

	return wr.String(), []any{"*.dev", filters.DeploymentID, filters.DisplayName}, err
}

func (s *Store) queryWithDomainName(filters ConfigFilters) (string, []any, error) {
	var wr bytes.Buffer

	err := s.selectConfigs.Execute(&wr, map[string]any{
		"join":    "INNER JOIN domains dm ON dm.env_id = dp.env_id",
		"where":   "dm.domain_name = $1 AND dm.domain_verified IS TRUE AND dp.deployment_id IS NOT NULL",
		"columns": "dm.custom_cert_value as cert_value, dm.custom_cert_key as cert_key, dm.domain_id",
	})

	return wr.String(), []any{strings.ToLower(filters.HostName)}, err
}

func (s *Store) queryWithEnvNameAndDisplayName(filters ConfigFilters) (string, []any, error) {
	var wr bytes.Buffer

	err := s.selectConfigs.Execute(&wr, map[string]any{
		"join": "",
		"where": `dp.deployment_id IS NOT NULL AND (
			(e.env_name = $2 AND LOWER(a.display_name) = LOWER($3)) OR 
			((e.env_id in (SELECT dm.env_id FROM domains dm WHERE dm.domain_name = $4 and dm.domain_verified IS TRUE)))
		)`,
	})

	return wr.String(), []any{"*.dev", filters.EnvName, filters.DisplayName, filters.HostName}, err
}

// ConfigsByDomain returns the hosting configurations for the given domain name.
// A domain can have multiple configurations, but their sum of percentage
// should always be 100.
func (s *Store) Configs(ctx context.Context, filters ConfigFilters) ([]*Config, error) {
	var query string
	var params []any
	var err error

	if filters.DeploymentID != 0 {
		query, params, err = s.queryWithDeploymentID(filters)
	} else if filters.DisplayName != "" {
		if filters.EnvName == "" {
			filters.EnvName = config.AppDefaultEnvironmentName
		}

		query, params, err = s.queryWithEnvNameAndDisplayName(filters)
	} else if filters.HostName != "" {
		query, params, err = s.queryWithDomainName(filters)
	} else {
		return nil, nil
	}

	if err != nil {
		slog.Errorf("error while creating query template: %v", err)
		return nil, err
	}

	return rowsToConfigs(s.Query(ctx, query, params...))
}

type Snippets []struct {
	Content  string                 `json:"content"`
	Location string                 `json:"location"`
	Prepend  bool                   `json:"prepend"`
	Rules    *buildconf.SnippetRule `json:"rules"`
}

func (s *Snippets) Scan(value any) error {
	if value != nil {
		return json.Unmarshal(value.([]byte), &s)
	}

	return nil
}

func rowsToConfigs(rows *sql.Rows, err error) ([]*Config, error) {
	if rows == nil || err != nil {
		return nil, err
	}

	cnfs := []*Config{}

	for rows.Next() {
		var buildConf []byte
		var buildManifest *deploy.BuildManifest
		var certKey string
		var certVal string
		var displayName string
		var envName string
		var tier string
		authwall := authwall.Config{}
		cnf := &Config{}
		err := rows.Scan(
			&cnf.AppID, &cnf.DeploymentID, &cnf.EnvID,
			&cnf.FunctionLocation, &cnf.StorageLocation,
			&cnf.APILocation, &cnf.APIPathPrefix,
			&buildManifest, &cnf.UpdatedAt, &buildConf,
			&cnf.Percentage, &certVal, &certKey, &cnf.DomainID,
			&authwall, &cnf.Snippets, &displayName, &envName, &tier,
			&cnf.BillingUserID,
		)

		if err != nil {
			return nil, err
		}

		if certKey != "" && certVal != "" {
			cnf.CertValue = utils.DecryptToString(certVal)
			cnf.CertKey = utils.DecryptToString(certKey)
		}

		customHeaders := []deploy.CustomHeader{}

		if buildConf != nil {
			data := buildconf.BuildConf{}

			if err := json.Unmarshal(buildConf, &data); err != nil {
				return nil, err
			}

			if data.ErrorFile != "" {
				data.ErrorFile = "/" + strings.TrimLeft(data.ErrorFile, "/")
			}

			if data.Headers != "" {
				customHeaders, err = deploy.ParseHeaders(data.Headers)

				// At this point simply log the error if it exists
				if err != nil {
					slog.Errorf("error while parsing custom headers: %s", err.Error())
				}
			}

			cnf.Redirects = data.Redirects
			cnf.ServerCmd = data.ServerCmd
			cnf.ErrorFile = data.ErrorFile
			cnf.EnvVariables = data.InterpolatedVars(
				buildconf.InterpolatedVarsOpts{
					DeploymentID: cnf.DeploymentID.String(),
					AppID:        cnf.AppID.String(),
					EnvID:        cnf.EnvID.String(),
					Env:          envName,
					DisplayName:  displayName,
				},
			)
		}

		if buildManifest != nil {
			staticFiles := StaticFileConfig{}

			// Backwards compatibility
			for _, v := range buildManifest.CDNFiles {
				staticFiles["/"+strings.TrimPrefix(strings.ToLower(v.Name), "/")] = &StaticFile{
					Headers:  deploy.ApplyHeaders(v.Name, NormalizeHeaders(v.Name, v.Headers), customHeaders),
					FileName: v.Name,
				}
			}

			for k, v := range buildManifest.StaticFiles {
				staticFiles[strings.ToLower(k)] = &StaticFile{
					Headers:  deploy.ApplyHeaders(k, v, customHeaders),
					FileName: k,
				}
			}

			cnf.Redirects = append(cnf.Redirects, buildManifest.Redirects...)
			cnf.StaticFiles = staticFiles
		}

		if authwall.Status != "" {
			cnf.AuthWall = authwall.Status
		}

		for _, sn := range cnf.Snippets {
			if sn.Rules != nil && sn.Rules.Path != "" {
				re, err := regexp2.Compile(sn.Rules.Path, regexp2.None)

				if err != nil {
					slog.Errorf("error while compiling regexp for snippet rule: %s", err.Error())
					continue
				}

				sn.Rules.PathCompiled = re
			}
		}

		if config.IsSelfHosted() {
			cnf.IsEnterprise = admin.CurrentLicense().Enterprise
		} else {
			cnf.IsEnterprise = tier == config.PackagePremium || tier == config.PackageUltimate
		}

		cnfs = append(cnfs, cnf)
	}

	if len(cnfs) == 0 {
		return nil, nil
	}

	return cnfs, nil
}

func NormalizeHeaders(fileName string, headers deploy.HeaderKeyValue) deploy.HeaderKeyValue {
	if headers == nil {
		return nil
	}

	normalized := make(map[string]string, len(headers))
	ext := path.Ext(fileName)

	for k, v := range headers {
		normalized[strings.ToLower(k)] = v
	}

	if normalized["content-type"] == "" {
		// Use built-in package to find the content-type
		if ext != "" {
			normalized["content-type"] = mime.TypeByExtension(ext)
		}

		// If it's not found, use the additional mime types
		if normalized["content-type"] == "" {
			normalized["content-type"] = deploy.AdditionalMimeTypesLower[ext]
		}

		// If it's still not found, default to `text/html`
		if normalized["content-type"] == "" {
			normalized["content-type"] = "text/html; charset=utf-8"
		}
	}

	return normalized
}
