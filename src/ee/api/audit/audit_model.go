package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

const (
	CreateAction string = "CREATE"
	UpdateAction string = "UPDATE"
	DeleteAction string = "DELETE"
)

const (
	TypeUser       string = "USER"
	TypeApp        string = "APP"
	TypeEnv        string = "ENV"
	TypeTeam       string = "TEAM"
	TypeDomain     string = "DOMAIN"
	TypeSnippet    string = "SNIPPET"
	TypeAuthWall   string = "AUTHWALL"
	TypeSchema     string = "SCHEMA"
	TypeDeployment string = "DEPLOYMENT"
)

// AuditData is the data extracted from a request context for auditing.
type AuditData struct {
	UserID      types.ID
	TeamID      types.ID
	AppID       types.ID
	EnvID       types.ID
	UserDisplay string
	TokenName   string
	Ctx         context.Context
}

// AuditContext can be implemented by any request context that cannot be
// directly imported here (e.g. to break import cycles). Go's structural typing
// means the implementor does not need to import this package explicitly.
type AuditContext interface {
	GetAuditData() AuditData
}

type DiffFields struct {
	AppName                  string                 `json:"appName,omitempty"`
	AppRepo                  string                 `json:"appRepo,omitempty"`
	EnvID                    string                 `json:"envId,omitempty"`
	EnvName                  string                 `json:"envName,omitempty"`
	EnvBranch                string                 `json:"envBranch,omitempty"`
	EnvBuildConfig           *buildconf.BuildConf   `json:"envBuildConfig,omitempty"`
	EnvAutoPublish           *bool                  `json:"envAutoPublish,omitempty"`
	EnvAutoDeploy            *bool                  `json:"envAutoDeploy,omitempty"`
	EnvAutoDeployBranches    string                 `json:"envAutoDeployBranches,omitempty"`
	EnvAutoDeployCommits     string                 `json:"envAutoDeployCommits,omitempty"`
	DomainName               string                 `json:"domainName,omitempty"`
	DomainCertValue          string                 `json:"domainCertValue,omitempty"`
	DomainCertKey            string                 `json:"domainCertKey,omitempty"`
	SnippetTitle             string                 `json:"snippetTitle,omitempty"`
	SnippetContent           string                 `json:"snippetContent,omitempty"`
	SnippetEnabled           *bool                  `json:"snippetEnabled,omitempty"`
	SnippetPrepend           *bool                  `json:"snippetPrepend,omitempty"`
	SnippetRules             *buildconf.SnippetRule `json:"snippetRules,omitempty"`
	SnippetLocation          string                 `json:"snippetLocation,omitempty"`
	Snippets                 []string               `json:"snippets,omitempty"`
	AuthWallStatus           string                 `json:"authWallStatus,omitempty"`
	AuthWallCreateLoginEmail string                 `json:"authWallCreateLoginEmail,omitempty"`
	AuthWallCreateLoginID    string                 `json:"authWallCreateLoginId,omitempty"`
	AuthWallDeleteLoginIDs   string                 `json:"authWallDeleteLoginIds,omitempty"`
	SchemaName               string                 `json:"schemaName,omitempty"`
	DeploymentID             string                 `json:"deploymentId,omitempty"`
	AutoPublished            *bool                  `json:"autoPublished,omitempty"`
}

type Diff struct {
	Old     DiffFields `json:"old"`
	New     DiffFields `json:"new"`
	changed *bool
}

// HasChanged returns whether the item changed or not.
func (d *Diff) HasChanged() bool {
	if d.changed == nil {
		changed := !reflect.DeepEqual(d.Old, d.New)
		d.changed = &changed
	}

	return *d.changed
}

type Audit struct {
	ID          types.ID   `json:"id,string"`
	UserID      types.ID   `json:"-"`
	TeamID      types.ID   `json:"teamId,string,omitempty"`
	AppID       types.ID   `json:"appId,string,omitempty"`
	EnvID       types.ID   `json:"envId,string,omitempty"`
	Action      string     `json:"action"`
	Diff        *Diff      `json:"diff,omitempty"`
	UserDisplay string     `json:"userDisplay,omitempty"`
	TokenName   string     `json:"tokenName,omitempty"`
	EnvName     string     `json:"envName,string"`
	Timestamp   utils.Unix `json:"timestamp"`

	ctx context.Context
}

// Bool returns the the pointer to the boolean value.
func Bool(b bool) *bool {
	return &b
}

// New creates a new Audit with the given context. Use this when there is no
// request context available (e.g. background jobs or auto-publish flows).
func New(ctx context.Context) *Audit {
	return &Audit{ctx: ctx}
}

func FromRequestContext(req any) *Audit {
	var ctx context.Context

	userID := types.ID(0)
	appID := types.ID(0)
	envID := types.ID(0)
	teamID := types.ID(0)
	userDisplay := ""
	tokenName := ""

	switch r := req.(type) {
	case *app.RequestContext:
		if r.User != nil {
			userID = r.User.ID
			userDisplay = r.User.Display()
		}

		if r.Token != nil {
			tokenName = r.Token.Name
		}

		if r.App != nil {
			teamID = r.App.TeamID
			appID = r.App.ID
		}

		if r.EnvID != 0 {
			envID = r.EnvID
		}

		ctx = r.Context()
	case *user.RequestContext:
		if r.User != nil {
			userID = r.User.ID
			userDisplay = r.User.Display()
		}

		ctx = r.Context()
	case *shttp.RequestContext:
		ctx = r.Context()
	case AuditContext:
		d := r.GetAuditData()
		userID, teamID, appID, envID = d.UserID, d.TeamID, d.AppID, d.EnvID
		userDisplay, tokenName, ctx = d.UserDisplay, d.TokenName, d.Ctx
	default:
		break
	}

	return &Audit{
		TeamID:      teamID,
		UserID:      userID,
		AppID:       appID,
		EnvID:       envID,
		UserDisplay: userDisplay,
		TokenName:   tokenName,
		ctx:         ctx,
	}
}

func (a *Audit) WithEnvID(envID types.ID) *Audit {
	a.EnvID = envID
	return a
}

func (a *Audit) WithAppID(appID types.ID) *Audit {
	a.AppID = appID
	return a
}

func (a *Audit) WithTeamID(teamID types.ID) *Audit {
	a.TeamID = teamID
	return a
}

func (a *Audit) WithTokenName(tokenName string) *Audit {
	a.TokenName = tokenName
	return a
}

func (a *Audit) WithAction(action, actionType string) *Audit {
	a.Action = fmt.Sprintf("%s:%s", action, actionType)
	return a
}

func (a *Audit) WithDiff(diff *Diff) *Audit {
	a.Diff = diff
	return a
}

func (a *Audit) Insert() error {
	// Validate that diffs are not equal. Otherwise there is no reason in storing a log.
	if a.Diff != nil && !a.Diff.HasChanged() {
		return nil
	}

	return NewStore().Log(a.ctx, a)
}

// ToMap transforms the struct into a map object. We're not using the
// `json` annotations because while marshaling an array of audit objects
// it doesn't behave as expected.
func (a Audit) ToMap() map[string]any {
	m := map[string]any{
		"id":          a.ID.String(),
		"appId":       a.AppID,
		"envId":       a.EnvID,
		"teamId":      a.TeamID,
		"action":      a.Action,
		"diff":        a.Diff,
		"userDisplay": a.UserDisplay,
		"tokenName":   a.TokenName,
		"envName":     a.EnvName,
	}

	if a.Timestamp.Valid {
		m["timestamp"] = a.Timestamp.UnixStr()
	}

	return m
}

// JSON transforms Audit object into a json string that is ready to be sent to the client.
func (a Audit) JSON() string {
	m := a.ToMap()
	b, _ := json.Marshal(m)
	return string(b)
}
