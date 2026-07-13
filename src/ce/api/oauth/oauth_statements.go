package oauth

import "fmt"

var tableAccessTokens = "user_access_tokens"

type userStatement struct {
	upsertToken    string
	selectAuthUser string
}

var ustmt = &userStatement{
	selectAuthUser: fmt.Sprintf(`
		SELECT
			display_name, account_uri, token_value, token_refresh,
			token_type, expire_at, personal_access_token
		FROM %s
		WHERE user_id = $1 AND provider = $2;
	`, tableAccessTokens),

	upsertToken: fmt.Sprintf(`
		INSERT INTO %s
			(user_id, display_name, account_uri, provider,
			 token_value, token_refresh, token_type, expire_at)
		VALUES
			($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT(user_id, provider) DO UPDATE
		SET
			display_name = $9,
			account_uri = $10,
			token_value = $11,
			token_refresh = $12,
			token_type = $13,
			expire_at = $14;
	`, tableAccessTokens),
}
