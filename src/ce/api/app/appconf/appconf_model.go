package appconf

import (
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/redirects"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

type StaticFile struct {
	FileName string            `json:"fileName,omitempty"`
	Headers  map[string]string `json:"headers,omitempty"`
}

type StaticFileConfig = map[string]*StaticFile

type Config struct {
	DeploymentID     types.ID              `json:"deploymentId,string"`
	AppID            types.ID              `json:"appId,string"`
	EnvID            types.ID              `json:"envId,string"`
	BillingUserID    types.ID              `json:"billingUserId,string,omitempty"`
	Domains          []string              `json:"domains"`
	ErrorFile        string                `json:"errorFile,omitempty"`
	StorageLocation  string                `json:"storageLocation,omitempty"`
	FunctionLocation string                `json:"functionLocation,omitempty"`
	APIPathPrefix    string                `json:"apiPathPrefix"`
	APILocation      string                `json:"apiLocation,omitempty"`
	ServerCmd        string                `json:"serverCmd,omitempty"`
	Percentage       float64               `json:"percentage"` // Percentage released: either 100 o 0
	Snippets         Snippets              `json:"snippets,omitempty"`
	UpdatedAt        utils.Unix            `json:"updatedAt"`
	Redirects        []redirects.Redirect  `json:"redirects,omitempty"`
	EnvVariables     map[string]string     `json:"envVariables,omitempty"`
	CertKey          string                `json:"certKey,omitempty"`
	CertValue        string                `json:"certValue,omitempty"`
	DomainID         types.ID              `json:"domainId,omitempty"`
	StaticFiles      StaticFileConfig      `json:"staticFiles,omitempty"`
	SKAuth           *buildconf.SKAuthConf `json:"-"`
	AuthWall         string                `json:"authWall,omitempty"`     // Whether to display an auth wall or not. Possible values: dev | all
	IsEnterprise     bool                  `json:"isEnterprise,omitempty"` // Whether the app is running in enterprise mode
}
