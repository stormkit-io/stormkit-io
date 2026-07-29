// Package bootstrap provisions a self-hosted instance non-interactively on
// first boot. It is used by the `install.sh --agent` flow to create the
// initial admin user and mint an owner-scoped API key, so an agent can
// manage the instance through the MCP endpoint without any dashboard steps.
package bootstrap

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/apikey"
	"github.com/stormkit-io/stormkit-io/src/ce/api/oauth"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stormkit-io/stormkit-io/src/lib/slog"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

// agentKeyName is the display name of the owner API key minted for agents.
const agentKeyName = "agent"

// Params configures a first-boot bootstrap.
type Params struct {
	AdminEmail    string
	AdminPassword string

	// AgentAPIKey is the raw token to store for the agent. When empty a
	// value is generated. Either way only the SHA-256 hash is persisted and
	// the raw value is surfaced once via Result.APIKey.
	AgentAPIKey string
}

// Result reports what the bootstrap did.
type Result struct {
	// Created is false when an admin already existed and the run was a no-op.
	Created bool

	// APIKey is the raw owner token, set only when Created is true.
	APIKey string
}

func (p Params) validate() error {
	if p.AdminEmail == "" || p.AdminPassword == "" {
		return fmt.Errorf("admin email and password are required")
	}

	if !utils.IsValidEmail(p.AdminEmail) {
		return fmt.Errorf("admin email is invalid")
	}

	if len(p.AdminPassword) < admin.MinAdminPasswordLength {
		return fmt.Errorf("admin password must be at least %d characters long", admin.MinAdminPasswordLength)
	}

	if p.AgentAPIKey != "" && !strings.HasPrefix(p.AgentAPIKey, "SK_") {
		return fmt.Errorf("agent api key must be prefixed with SK_")
	}

	return nil
}

// Run creates the initial admin user and mints an owner-scoped API key so an
// agent can manage the instance immediately. It is idempotent: if an admin
// already exists it returns Result{Created: false} without making changes.
func Run(ctx context.Context, p Params) (*Result, error) {
	if err := p.validate(); err != nil {
		return nil, err
	}

	cfg, err := admin.Store().Config(ctx)

	if err != nil {
		return nil, err
	}

	if cfg.AdminUserConfig != nil {
		return &Result{Created: false}, nil
	}

	userStore := user.NewStore()

	// AdminUserConfig is only set by the email/password admin path, so on its
	// own it misses instances first provisioned via OAuth or magic link. Bail
	// when a user that is not our own admin already exists, so bootstrap never
	// grafts a new privileged admin onto an already-populated instance.
	//
	// A partial run of *this* bootstrap (user/key created, but a later step
	// failed before AdminUserConfig was written) leaves only our admin behind.
	// Excluding it from the count means such a run is retried on the next boot
	// rather than short-circuited into a permanently unconfigured instance.
	adminUser, err := userStore.UserByEmail(ctx, []string{p.AdminEmail})

	if err != nil {
		return nil, err
	}

	count, err := userStore.SelectTotalUsers(ctx)

	if err != nil {
		return nil, err
	}

	foreignUsers := count

	if adminUser != nil {
		foreignUsers--
	}

	if foreignUsers > 0 {
		return &Result{Created: false}, nil
	}

	// MustUser atomically creates the admin, a default team and an owner
	// membership, so the minted key resolves against a real team. It returns
	// the existing admin unchanged when a prior partial run already created it.
	usr, err := userStore.MustUser(oauth.NewAdminUser(p.AdminEmail))

	if err != nil {
		return nil, err
	}

	token := p.AgentAPIKey

	if token == "" {
		token = apikey.GenerateTokenValue()
	}

	keyStore := apikey.NewStore()

	// Skip minting when a prior partial run already stored this key, so a retry
	// neither duplicates it nor trips the unique constraint on the token hash.
	existingKey, err := keyStore.APIKey(ctx, token)

	if err != nil {
		return nil, err
	}

	if existingKey == nil {
		key := &apikey.Token{
			UserID: usr.ID,
			Name:   agentKeyName,
			Scope:  apikey.SCOPE_USER,
			Value:  token,
		}

		if err := keyStore.AddAPIKey(ctx, key); err != nil {
			return nil, err
		}
	}

	// AdminUserConfig is written last, so it doubles as the completion marker:
	// until it is set, a partial run above is finished on the next boot instead
	// of short-circuiting on a state that can never log in. Every step above is
	// idempotent, so repeating them on retry is safe.
	adminCfg, err := admin.NewAdminUserConfig(p.AdminEmail, p.AdminPassword)

	if err != nil {
		return nil, err
	}

	cfg.AdminUserConfig = adminCfg

	if err := admin.Store().UpsertConfig(ctx, cfg); err != nil {
		return nil, err
	}

	return &Result{Created: true, APIKey: token}, nil
}

// FromEnv runs the bootstrap using STORMKIT_ADMIN_EMAIL,
// STORMKIT_ADMIN_PASSWORD and STORMKIT_AGENT_API_KEY. It is a no-op unless the
// instance is self-hosted and both admin credentials are set, so it is safe to
// call unconditionally at startup. The raw API key is never logged; it is
// surfaced to the operator by install.sh, which owns the value.
func FromEnv(ctx context.Context) {
	if !config.IsSelfHosted() {
		return
	}

	email := os.Getenv("STORMKIT_ADMIN_EMAIL")
	password := os.Getenv("STORMKIT_ADMIN_PASSWORD")

	if email == "" || password == "" {
		return
	}

	// A generated key would only live in memory and never reach the operator,
	// so require it to be supplied here rather than silently minting an
	// unrecoverable one. install.sh always sets it.
	apiKey := os.Getenv("STORMKIT_AGENT_API_KEY")

	if apiKey == "" {
		slog.Errorf("agent bootstrap: STORMKIT_AGENT_API_KEY is required when admin credentials are set; skipping")
		return
	}

	res, err := Run(ctx, Params{
		AdminEmail:    email,
		AdminPassword: password,
		AgentAPIKey:   apiKey,
	})

	if err != nil {
		slog.Errorf("agent bootstrap failed: %v", err)
		return
	}

	if res.Created {
		slog.Infof("agent bootstrap: created admin user %s and minted owner API key", email)
	}
}
