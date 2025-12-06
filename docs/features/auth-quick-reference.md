# Auth Feature - Quick Reference Guide

## For Stormkit Users (Developers)

### Setting Up OAuth Authentication

#### Step 1: Get OAuth Credentials

**For Google:**
1. Go to [Google Cloud Console](https://console.cloud.google.com/)
2. Create a new project or select existing
3. Enable Google Identity services (OAuth 2.0 API)
4. Go to "Credentials" → "Create Credentials" → "OAuth 2.0 Client ID"
5. Set authorized redirect URIs (you'll get this from Stormkit)
6. Copy Client ID and Client Secret

**For X (formerly Twitter):**
1. Go to [X Developer Portal](https://developer.x.com/)
2. Create an app
3. Go to "Keys and tokens"
4. Copy API Key (Client ID) and API Secret (Client Secret)

#### Step 2: Configure in Stormkit

1. Open your app in Stormkit Dashboard
2. Click "Auth" tab in sidebar
3. Click "Configure Google" (or other provider)
4. Enter your Client ID and Client Secret
5. Copy the auto-generated redirect URI
6. Add this redirect URI to your OAuth provider settings
7. Save configuration

#### Step 3: Integrate in Your App

**HTML/JavaScript:**
```html
<button id="login">Login with Google</button>

<script>
  const APP_ID = 'your-app-id';
  const API_BASE = 'https://api.stormkit.io';
  
  document.getElementById('login').addEventListener('click', () => {
    const redirectUri = `${window.location.origin}/auth/callback`;
    window.location.href = 
      `${API_BASE}/public/auth/${APP_ID}/google/login?redirect_uri=${encodeURIComponent(redirectUri)}`;
  });
</script>
```

**React:**
```jsx
import { useState, useEffect } from 'react';

const APP_ID = 'your-app-id';
const API_BASE = 'https://api.stormkit.io';

function App() {
  const [user, setUser] = useState(null);

  useEffect(() => {
    const token = localStorage.getItem('auth_token');
    if (token) {
      fetchUser(token);
    }
  }, []);

  const fetchUser = async (token) => {
    const res = await fetch(`${API_BASE}/public/auth/${APP_ID}/user`, {
      headers: { 'Authorization': `Bearer ${token}` }
    });
    if (res.ok) {
      setUser(await res.json());
    }
  };

  const login = () => {
    const redirectUri = `${window.location.origin}/auth/callback`;
    window.location.href = 
      `${API_BASE}/public/auth/${APP_ID}/google/login?redirect_uri=${encodeURIComponent(redirectUri)}`;
  };

  const logout = async () => {
    const token = localStorage.getItem('auth_token');
    await fetch(`${API_BASE}/public/auth/${APP_ID}/logout`, {
      method: 'POST',
      headers: { 'Authorization': `Bearer ${token}` }
    });
    localStorage.removeItem('auth_token');
    setUser(null);
  };

  if (!user) {
    return <button onClick={login}>Login with Google</button>;
  }

  return (
    <div>
      <p>Welcome, {user.displayName}!</p>
      <button onClick={logout}>Logout</button>
    </div>
  );
}
```

**Next.js:**
```javascript
// pages/api/auth/callback.js
export default async function handler(req, res) {
  const { token } = req.query;
  
  if (token) {
    // Verify token and set cookie
    res.setHeader('Set-Cookie', `auth_token=${token}; HttpOnly; Secure; SameSite=Strict`);
    res.redirect('/dashboard');
  } else {
    res.redirect('/login?error=auth_failed');
  }
}
```

### API Reference

#### Login
```
GET /public/auth/:appId/:provider/login
Query Params:
  - redirect_uri: URL to redirect after authentication
  - state: Optional CSRF token
```

#### Get User
```
GET /public/auth/:appId/user
Headers:
  - Authorization: Bearer <token>
Response:
{
  "user": {
    "id": "uuid",
    "email": "user@example.com",
    "displayName": "John Doe",
    "avatarUrl": "https://...",
    "provider": "google"
  }
}
```

#### Refresh Token
```
POST /public/auth/:appId/refresh
Body:
{
  "refreshToken": "xxx"
}
Response:
{
  "accessToken": "xxx",
  "refreshToken": "xxx",
  "expiresAt": "2025-01-01T00:00:00Z"
}
```

#### Logout
```
POST /public/auth/:appId/logout
Headers:
  - Authorization: Bearer <token>
```

## For Stormkit Developers

### Adding a New OAuth Provider

#### Step 1: Create Provider Implementation

Create `src/ce/api/app/appauth/providers/google.go`:

```go
package providers

import (
    "context"
    "encoding/json"
    "fmt"
    "golang.org/x/oauth2"
    "golang.org/x/oauth2/google"
)

type GoogleProvider struct {
    ClientID     string
    ClientSecret string
    RedirectURL  string
    Scopes       []string
}

func (p *GoogleProvider) GetAuthURL(state string) string {
    config := p.getOAuthConfig()
    return config.AuthCodeURL(state, oauth2.AccessTypeOffline)
}

func (p *GoogleProvider) ExchangeCode(code string) (*oauth2.Token, error) {
    config := p.getOAuthConfig()
    return config.Exchange(context.Background(), code)
}

func (p *GoogleProvider) GetUserInfo(token *oauth2.Token) (*AuthUser, error) {
    client := p.getOAuthConfig().Client(context.Background(), token)
    resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    var userInfo struct {
        ID            string `json:"id"`
        Email         string `json:"email"`
        VerifiedEmail bool   `json:"verified_email"`
        Name          string `json:"name"`
        Picture       string `json:"picture"`
    }

    if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
        return nil, err
    }

    return &AuthUser{
        ProviderUserID: userInfo.ID,
        Email:          userInfo.Email,
        EmailVerified:  userInfo.VerifiedEmail,
        DisplayName:    userInfo.Name,
        AvatarURL:      userInfo.Picture,
    }, nil
}

func (p *GoogleProvider) GetProviderType() string {
    return "google"
}

func (p *GoogleProvider) GetDefaultScopes() []string {
    return []string{"email", "profile"}
}

func (p *GoogleProvider) getOAuthConfig() *oauth2.Config {
    scopes := p.Scopes
    if len(scopes) == 0 {
        scopes = p.GetDefaultScopes()
    }

    return &oauth2.Config{
        ClientID:     p.ClientID,
        ClientSecret: p.ClientSecret,
        RedirectURL:  p.RedirectURL,
        Scopes:       scopes,
        Endpoint:     google.Endpoint,
    }
}
```

#### Step 2: Register Provider in Factory

Add to `src/ce/api/app/appauth/providers/provider_interface.go`:

```go
func NewProvider(providerType, clientID, clientSecret, redirectURL string, scopes []string) (OAuthProvider, error) {
    switch providerType {
    case "google":
        return &GoogleProvider{
            ClientID:     clientID,
            ClientSecret: clientSecret,
            RedirectURL:  redirectURL,
            Scopes:       scopes,
        }, nil
    case "x":
        return &XProvider{...}, nil
    default:
        return nil, fmt.Errorf("unsupported provider: %s", providerType)
    }
}
```

#### Step 3: Add Tests

Create `src/ce/api/app/appauth/providers/google_test.go`:

```go
package providers

import (
    "testing"
)

func TestGoogleProvider_GetAuthURL(t *testing.T) {
    provider := &GoogleProvider{
        ClientID:     "test-client-id",
        ClientSecret: "test-secret",
        RedirectURL:  "http://localhost/callback",
    }

    url := provider.GetAuthURL("test-state")
    
    if url == "" {
        t.Error("Expected non-empty auth URL")
    }
    
    // Add more assertions
}
```

### Database Queries

#### Get App Providers
```sql
SELECT 
    provider_id, 
    provider_type, 
    provider_name, 
    client_id,
    redirect_uri,
    scopes,
    enabled
FROM skitapi.app_auth_providers
WHERE app_id = $1
ORDER BY created_at DESC;
```

#### Create Session
```sql
INSERT INTO skitapi.app_auth_sessions (
    auth_user_id,
    access_token,
    refresh_token,
    expires_at,
    user_agent,
    ip_address
) VALUES ($1, $2, $3, $4, $5, $6)
RETURNING session_id;
```

### Testing

#### Unit Tests
```bash
cd src/ce/api/app/appauth
go test -v ./...
```

#### Integration Tests
```bash
# Start test database
docker compose up -d db

# Run integration tests
go test -v -tags=integration ./...
```

### Security Checklist

- [ ] Client secrets encrypted with AES-256
- [ ] JWT tokens signed with secure key
- [ ] State parameter validated on callback
- [ ] Rate limiting enabled on all auth endpoints
- [ ] Redirect URIs validated
- [ ] HTTPS enforced in production
- [ ] CORS properly configured
- [ ] Session expiration implemented
- [ ] Refresh token rotation enabled
- [ ] SQL injection prevention (parameterized queries)
- [ ] XSS prevention (proper escaping)
- [ ] CSRF protection (state token)

## Common Issues & Solutions

### Issue: "Invalid redirect URI"
**Solution:** Make sure the redirect URI in your OAuth provider settings exactly matches the one shown in Stormkit dashboard.

### Issue: "Token expired"
**Solution:** Implement token refresh using the `/refresh` endpoint before the token expires.

### Issue: "CORS error"
**Solution:** Make sure your app domain is properly configured in Stormkit. Auth endpoints have CORS enabled for configured domains.

### Issue: "Provider not enabled"
**Solution:** Check that the provider is enabled in the Stormkit dashboard under Auth settings.

## Best Practices

1. **Token Storage**: Store tokens in httpOnly cookies for web apps, secure storage for mobile apps
2. **Error Handling**: Always handle authentication errors gracefully
3. **User Experience**: Show loading states during OAuth redirects
4. **Token Refresh**: Implement automatic token refresh before expiration
5. **Logout**: Always call the logout endpoint when user logs out
6. **Testing**: Test authentication flow in different browsers
7. **Security**: Never expose client secrets in frontend code
8. **Monitoring**: Monitor auth failure rates in Stormkit dashboard

## Resources

- [OAuth 2.0 Spec](https://oauth.net/2/)
- [Google OAuth Documentation](https://developers.google.com/identity/protocols/oauth2)
- [X OAuth Documentation](https://developer.x.com/en/docs/authentication/oauth-2-0)
- [Stormkit Documentation](https://stormkit.io/docs)
