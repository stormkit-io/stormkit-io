package buildconf

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"text/template"

	"github.com/stormkit-io/stormkit-io/src/lib/database"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
	"gopkg.in/guregu/null.v3"
)

var tableDomains = "domains"

var dstmt = struct {
	selectDomains    string
	insertDomain     string
	deleteDomains    string
	verifyDomain     string
	updateDomainCert string
	updateLastPing   string
}{
	selectDomains: `
		SELECT
			d.domain_id, d.app_id, d.env_id, d.domain_name,
			d.domain_verified, d.domain_verified_at, d.domain_token,
			d.custom_cert_value, d.custom_cert_key, d.last_ping
		FROM
			domains d
		WHERE
			{{ .where }}
		ORDER BY
			d.domain_id ASC
		LIMIT
			{{ or .limit 1 }};
	`,

	insertDomain: `
		INSERT INTO domains
			(app_id, env_id, domain_name, domain_verified, domain_verified_at, domain_token)
		VALUES
			($1, $2, $3, $4, $5, $6)
		RETURNING
			domain_id;
	`,

	deleteDomains: `
		DELETE FROM domains WHERE {{ .where }};
	`,

	verifyDomain: `
		UPDATE domains SET domain_verified = TRUE WHERE domain_id = $1;
	`,

	updateDomainCert: `
		UPDATE domains SET custom_cert_value = $1, custom_cert_key = $2 WHERE domain_id = $3;
	`,

	updateLastPing: `
		UPDATE
			domains AS d
		SET
			last_ping = (v.last_ping_info)::jsonb
		FROM
			(VALUES {{ generateValues 2 (len .) }}) AS v(domain_id, last_ping_info)
		WHERE
			(v.domain_id)::integer = d.domain_id;
	`,
}

// Store represents a store for the deployments and deployment logs.
type DStore struct {
	*database.Store
	selectTmpl  *template.Template
	deleteTmpl  *template.Template
	updateBatch *template.Template
}

// NewStore returns a store instance.
func DomainStore() *DStore {
	return &DStore{
		Store: database.NewStore(),
		updateBatch: template.Must(
			template.New("batch_update_last_ping").
				Funcs(template.FuncMap{"generateValues": utils.GenerateValues}).
				Parse(dstmt.updateLastPing),
		),
		selectTmpl: template.Must(
			template.New("selectDomains").
				Parse(dstmt.selectDomains),
		),
		deleteTmpl: template.Must(
			template.New("deleteDomains").
				Parse(dstmt.deleteDomains),
		),
	}
}

func (s *DStore) selectDomain(ctx context.Context, data map[string]any, params ...any) (*DomainModel, error) {
	var wr bytes.Buffer

	domain := &DomainModel{}

	if err := s.selectTmpl.Execute(&wr, data); err != nil {
		return nil, err
	}

	var customCertVal null.String
	var customCertKey null.String

	row, err := s.QueryRow(ctx, wr.String(), params...)

	if err != nil {
		return nil, err
	}

	err = row.Scan(
		&domain.ID, &domain.AppID, &domain.EnvID,
		&domain.Name, &domain.Verified,
		&domain.VerifiedAt, &domain.Token,
		&customCertVal, &customCertKey,
		&domain.LastPing,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}

		return nil, err
	}

	if customCertVal.Valid && customCertKey.Valid {
		domain.CustomCert = &CustomCert{
			Value: utils.DecryptToString(customCertVal.ValueOrZero()),
			Key:   utils.DecryptToString(customCertKey.ValueOrZero()),
		}
	}

	return domain, nil
}

// DomainByID returns a domain by it's ID.
func (s *DStore) DomainByID(ctx context.Context, domainID types.ID) (*DomainModel, error) {
	data := map[string]any{
		"where": "d.domain_id = $1",
	}

	return s.selectDomain(ctx, data, domainID)
}

// DomainByName returns a domain by it's name.
func (s *DStore) DomainByName(ctx context.Context, domainName string) (*DomainModel, error) {
	data := map[string]any{
		"where": "d.domain_name = $1",
	}

	return s.selectDomain(ctx, data, domainName)
}

type DomainFilters struct {
	EnvID       types.ID
	AfterID     types.ID
	DomainName  string // Used for fuzzy search
	Limit       int
	ModInterval int  // The second argument for MOD() fn in PostgreSQL
	ModID       *int // The value of the MOD() fn
	Verified    *bool
}

// Domains returns a list of domains by their environment id.
func (s *DStore) Domains(ctx context.Context, filters DomainFilters) ([]*DomainModel, error) {
	var wr bytes.Buffer

	where := []string{}
	params := []any{}

	if filters.EnvID != 0 {
		where = append(where, "d.env_id = $1")
		params = append(params, filters.EnvID)
	}

	if filters.Verified != nil {
		params = append(params, *filters.Verified)
		where = append(where, fmt.Sprintf("d.domain_verified = $%d", len(params)))
	}

	if filters.AfterID != 0 {
		params = append(params, filters.AfterID)
		where = append(where, fmt.Sprintf("d.domain_id > $%d", len(params)))
	}

	if filters.ModID != nil && filters.ModInterval != 0 {
		params = append(params, *filters.ModID)
		where = append(where, fmt.Sprintf("MOD(d.domain_id, %d) = $%d", filters.ModInterval, len(params)))
	}

	if filters.DomainName != "" {
		params = append(params, "%"+filters.DomainName+"%")
		where = append(where, fmt.Sprintf("d.domain_name ILIKE $%d", len(params)))
	}

	limit := 100

	if filters.Limit > 0 {
		limit = filters.Limit
	}

	data := map[string]any{
		"where": strings.Join(where, " AND "),
		"limit": limit + 1,
	}

	if err := s.selectTmpl.Execute(&wr, data); err != nil {
		return nil, err
	}

	domains := []*DomainModel{}
	rows, err := s.Query(ctx, wr.String(), params...)

	if err != nil || rows == nil {
		return nil, err
	}

	defer rows.Close()

	for rows.Next() {
		domain := &DomainModel{}

		var customCertVal null.String
		var customCertKey null.String

		err := rows.Scan(
			&domain.ID, &domain.AppID, &domain.EnvID,
			&domain.Name, &domain.Verified,
			&domain.VerifiedAt, &domain.Token,
			&customCertVal, &customCertKey, &domain.LastPing,
		)

		if err != nil {
			return nil, err
		}

		if customCertVal.Valid && customCertKey.Valid {
			domain.CustomCert = &CustomCert{
				Value: utils.DecryptToString(customCertVal.ValueOrZero()),
				Key:   utils.DecryptToString(customCertKey.ValueOrZero()),
			}
		}

		domains = append(domains, domain)
	}

	return domains, nil
}

// Insert inserts a new domain record.
func (s *DStore) Insert(ctx context.Context, domain *DomainModel) error {
	row, err := s.QueryRow(
		ctx,
		dstmt.insertDomain,
		domain.AppID,
		domain.EnvID,
		domain.Name,
		domain.Verified,
		domain.VerifiedAt,
		domain.Token,
	)

	if err != nil {
		return err
	}

	return row.Scan(&domain.ID)
}

type DeleteDomainArgs struct {
	DomainID types.ID
	EnvID    types.ID
	AppID    types.ID
}

// DeleteDomain removes the associated domain information from the environment.
func (s *DStore) DeleteDomain(ctx context.Context, args DeleteDomainArgs) error {
	var where string
	var param any
	var wr bytes.Buffer

	if args.DomainID != 0 {
		where = "domain_id = $1"
		param = args.DomainID
	} else if args.EnvID != 0 {
		where = "env_id = $1"
		param = args.EnvID
	} else if args.AppID != 0 {
		where = "app_id = $1"
		param = args.AppID
	}

	if where == "" {
		return errors.New("invalid argument received: expecting one of domain_id, env_id or app_id fields")
	}

	data := map[string]any{
		"where": where,
	}

	if err := s.deleteTmpl.Execute(&wr, data); err != nil {
		return err
	}

	_, err := s.Exec(ctx, wr.String(), param)
	return err
}

// UpdateDomainCert updates the given domain TLS settings.
func (s *DStore) UpdateDomainCert(ctx context.Context, domain *DomainModel) error {
	var cert null.String
	var key null.String

	if domain.CustomCert != nil {
		cert = null.StringFrom(utils.EncryptToString(domain.CustomCert.Value))
		key = null.StringFrom(utils.EncryptToString(domain.CustomCert.Key))
	}

	_, err := s.Exec(ctx, dstmt.updateDomainCert, cert, key, domain.ID)
	return err
}

// VerifyDomain updates the domain record and sets the verified column as true.
func (s *DStore) VerifyDomain(ctx context.Context, domainID types.ID) error {
	_, err := s.Exec(ctx, dstmt.verifyDomain, domainID)
	return err
}

// UpdateLastPing updates the ping information for the given domains.
func (s *DStore) UpdateLastPing(ctx context.Context, res []PingResult) error {
	var qb strings.Builder

	if err := s.updateBatch.Execute(&qb, res); err != nil {
		return err
	}

	params := []any{}

	for _, result := range res {
		data, _ := json.Marshal(result)
		params = append(params, result.DomainID, data)
	}

	_, err := s.Exec(ctx, qb.String(), params...)
	return err
}
