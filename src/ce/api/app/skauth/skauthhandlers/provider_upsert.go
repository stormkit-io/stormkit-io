package skauthhandlers

import (
	"context"
	"strings"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/appcache"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/skauth"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttperr"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

// UpsertProviderParams carries everything UpsertProvider needs. Env is the
// environment being configured; it is mutated when an auth config has to be
// created on the fly.
type UpsertProviderParams struct {
	Env   *buildconf.Env
	AppID types.ID
	Data  AuthUpsertRequest
}

// UpsertProvider validates and stores a single sign-in provider. It is shared
// by the dashboard handler, the public API handler and the MCP tool. On
// validation failure it returns a *shttperr.ValidationError, which shttp.Error
// renders as a 400 with the field errors.
func UpsertProvider(ctx context.Context, p UpsertProviderParams) error {
	env := p.Env

	if env.SchemaConf == nil {
		return &shttperr.ValidationError{Errors: map[string]string{"schema": "Schema configuration is not set for this environment. Please configure it first."}}
	}

	if err := ensureAuthConf(ctx, env); err != nil {
		return err
	}

	data := p.Data
	data.ProviderName = normalizeProviderName(data.ProviderName)

	existing, err := skauth.NewStore().Provider(ctx, env.ID, data.ProviderName)

	if err != nil {
		return err
	}

	providerData, err := providerDataFor(providerDataForParams{Data: data, Existing: existing})

	if err != nil {
		return err
	}

	return saveProvider(ctx, saveProviderParams{
		Env:    env,
		AppID:  p.AppID,
		Name:   data.ProviderName,
		Status: resolveStatus(data.Status, existing),
		Data:   providerData,
	})
}

// resolveStatus applies patch semantics to the enabled flag: an omitted status
// keeps whatever the provider already had, so a caller rotating a client
// secret cannot disable a live provider by not mentioning it. A provider being
// created for the first time defaults to enabled.
func resolveStatus(supplied *bool, existing *skauth.Provider) bool {
	if supplied != nil {
		return *supplied
	}

	if existing != nil {
		return existing.Status
	}

	return true
}

// normalizeProviderName maps what a caller typed onto the canonical provider
// id. Separators are stripped so the natural spellings of "magic-link" and
// "magic_link" both reach skauth.ProviderMagicLink ("magiclink").
func normalizeProviderName(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))

	return strings.NewReplacer("-", "", "_", "", " ", "").Replace(name)
}

// ensureAuthConf creates a default auth configuration when the environment has
// none, so that enabling a provider is enough to get a working sign-in.
func ensureAuthConf(ctx context.Context, env *buildconf.Env) error {
	if env.AuthConf != nil {
		return nil
	}

	env.AuthConf = &buildconf.SKAuthConf{
		TTL:        7 * 24 * 60,
		SuccessURL: "/",
		Secret:     utils.RandomToken(128),
		Status:     true,
	}

	if err := buildconf.NewStore().SaveAuthConf(ctx, env.ID, env.AuthConf); err != nil {
		return err
	}

	return appcache.Service().Reset(env.ID)
}

type providerDataForParams struct {
	Data     AuthUpsertRequest
	Existing *skauth.Provider
}

// providerDataFor validates the provider-specific fields and returns the data
// to store. Email providers carry a from address; everything else is OAuth and
// needs a client ID and secret.
func providerDataFor(p providerDataForParams) (skauth.ProviderData, error) {
	data := p.Data

	switch data.ProviderName {
	case skauth.ProviderEmail, skauth.ProviderMagicLink:
		fromAddress := strings.TrimSpace(data.FromAddress)

		// An omitted from address keeps the stored one, so a caller that cannot
		// read it back can still toggle the provider.
		if fromAddress == "" && p.Existing != nil {
			fromAddress = p.Existing.Data.FromAddress
		}

		if data.ProviderName == skauth.ProviderMagicLink && fromAddress == "" {
			return skauth.ProviderData{}, &shttperr.ValidationError{Errors: map[string]string{"fromAddress": "From address is required"}}
		}

		return skauth.ProviderData{FromAddress: fromAddress}, nil
	}

	// An omitted or placeholder secret keeps the stored one, so a client that
	// never sees the secret can still toggle the provider.
	if data.ClientSecret == "" || data.ClientSecret == ClientSecretPlaceholder {
		if p.Existing != nil && p.Existing.Data.ClientSecret != "" {
			data.ClientSecret = p.Existing.Data.ClientSecret
		}
	}

	// An omitted client ID keeps the stored one, for the same reason.
	if data.ClientID == "" && p.Existing != nil {
		data.ClientID = p.Existing.Data.ClientID
	}

	if data.ClientID == "" {
		return skauth.ProviderData{}, &shttperr.ValidationError{Errors: map[string]string{"clientId": "Client ID is required"}}
	}

	if data.ClientSecret == "" {
		return skauth.ProviderData{}, &shttperr.ValidationError{Errors: map[string]string{"clientSecret": "Client Secret is required"}}
	}

	return skauth.ProviderData{
		ClientID:     data.ClientID,
		ClientSecret: data.ClientSecret,
	}, nil
}

type saveProviderParams struct {
	Env    *buildconf.Env
	AppID  types.ID
	Name   string
	Status bool
	Data   skauth.ProviderData
}

// saveProvider creates the auth table (idempotent) and upserts the provider.
func saveProvider(ctx context.Context, p saveProviderParams) error {
	migrationStore, err := p.Env.SchemaConf.Store(buildconf.SchemaAccessTypeMigrations)

	if err != nil {
		return err
	}

	defer migrationStore.Close()

	// This is idempotent - if the table already exists, no error is returned.
	if err := migrationStore.CreateAuthTable(ctx); err != nil {
		return err
	}

	provider := &skauth.Provider{
		Name:   p.Name,
		Status: p.Status,
		Data:   p.Data,
	}

	if !provider.Supported() {
		return &shttperr.ValidationError{Errors: map[string]string{"providerName": "Invalid provider"}}
	}

	return skauth.NewStore().SaveProvider(ctx, skauth.SaveProviderArgs{
		EnvID:    p.Env.ID,
		AppID:    p.AppID,
		Provider: provider,
	})
}
