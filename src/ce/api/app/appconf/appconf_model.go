package appconf

import (
	"strings"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/deploy"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/redirects"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

type StaticFile struct {
	FileName string            `json:"fileName,omitempty"`
	Headers  map[string]string `json:"headers,omitempty"`
}

type StaticFileConfig = map[string]*StaticFile

// ReservedPathPrefix is the deployment folder Stormkit reserves for files it
// consumes itself, matching the /_stormkit routes it serves.
const ReservedPathPrefix = "/_stormkit/"

// AddStaticFile files a deployment file under its lowercased path, in
// ReservedFiles when it sits under ReservedPathPrefix and in StaticFiles
// otherwise.
func (c *Config) AddStaticFile(path string, file *StaticFile) {
	path = strings.ToLower(path)

	if strings.HasPrefix(path, ReservedPathPrefix) {
		if c.ReservedFiles == nil {
			c.ReservedFiles = StaticFileConfig{}
		}

		c.ReservedFiles[path] = file

		return
	}

	if c.StaticFiles == nil {
		c.StaticFiles = StaticFileConfig{}
	}

	c.StaticFiles[path] = file
}

type Config struct {
	DeploymentID     types.ID             `json:"deploymentId,string"`
	AppID            types.ID             `json:"appId,string"`
	EnvID            types.ID             `json:"envId,string"`
	BillingUserID    types.ID             `json:"billingUserId,string,omitempty"`
	Domains          []string             `json:"domains"`
	ErrorFile        string               `json:"errorFile,omitempty"`
	Markdown         bool                 `json:"markdown,omitempty"`
	MarkdownConvert  bool                 `json:"markdownConvert,omitempty"`
	StorageLocation  string               `json:"storageLocation,omitempty"`
	FunctionLocation string               `json:"functionLocation,omitempty"`
	APIPathPrefix    string               `json:"apiPathPrefix"`
	APILocation      string               `json:"apiLocation,omitempty"`
	ServerCmd        string               `json:"serverCmd,omitempty"`
	IsPublished      bool                 `json:"isPublished"` // Whether the environment is serving this deployment
	Snippets         Snippets             `json:"snippets,omitempty"`
	UpdatedAt        utils.Unix           `json:"updatedAt"`
	Redirects        []redirects.Redirect `json:"redirects,omitempty"`
	EnvVariables     map[string]string    `json:"envVariables,omitempty"`
	CertKey          string               `json:"certKey,omitempty"`
	CertValue        string               `json:"certValue,omitempty"`
	DomainID         types.ID             `json:"domainId,omitempty"`
	StaticFiles      StaticFileConfig     `json:"staticFiles,omitempty"`

	// ReservedFiles holds the deployment's files under ReservedPathPrefix.
	// Stormkit reads them itself (e.g. the OAuth sign-in template) and never
	// serves them at their own path.
	ReservedFiles StaticFileConfig `json:"reservedFiles,omitempty"`

	CustomHeaders []deploy.CustomHeader `json:"-"` // Parsed user-configured header rules; matched against request path at response time.
	SKAuth        *buildconf.SKAuthConf `json:"-"`
	AuthWall      string                `json:"authWall,omitempty"` // Whether to display an auth wall or not. Possible values: dev | all

	// AuthWallPublicPaths skip the auth wall. See authwall.IsPublicPath.
	AuthWallPublicPaths []string `json:"authWallPublicPaths,omitempty"`

	IsEnterprise bool `json:"isEnterprise,omitempty"` // Whether the app is running in enterprise mode

	// AnalyticsExcluded mirrors the domain's opt-out. Requests served through
	// this host are still access-logged, but never reach visitor analytics.
	AnalyticsExcluded bool `json:"analyticsExcluded,omitempty"`
}
