# Stormkit Auth Architecture Diagram

## Component Overview

```
┌─────────────────────────────────────────────────────────────────────────┐
│                          Client Application                              │
│  ┌──────────────────────────────────────────────────────────────────┐  │
│  │  User clicks "Login with Google"                                 │  │
│  └──────────────────────────────────────────────────────────────────┘  │
└────────────────────────────────────┬────────────────────────────────────┘
                                     │
                                     │ 1. Redirect to Stormkit Auth
                                     │
                                     ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                     Stormkit API (Auth Endpoints)                        │
│  ┌──────────────────────────────────────────────────────────────────┐  │
│  │  GET /public/auth/:appId/:provider/login                         │  │
│  │  - Validates app and provider configuration                      │  │
│  │  - Generates state token for CSRF protection                     │  │
│  │  - Redirects to OAuth provider                                   │  │
│  └──────────────────────────────────────────────────────────────────┘  │
└────────────────────────────────────┬────────────────────────────────────┘
                                     │
                                     │ 2. Redirect to OAuth Provider
                                     │
                                     ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                     OAuth Provider (Google, X, etc.)                     │
│  ┌──────────────────────────────────────────────────────────────────┐  │
│  │  User authenticates and grants permissions                       │  │
│  └──────────────────────────────────────────────────────────────────┘  │
└────────────────────────────────────┬────────────────────────────────────┘
                                     │
                                     │ 3. Redirect back with auth code
                                     │
                                     ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                     Stormkit API (Callback Handler)                      │
│  ┌──────────────────────────────────────────────────────────────────┐  │
│  │  GET /public/auth/:appId/:provider/callback                      │  │
│  │  - Validates state token                                         │  │
│  │  - Exchanges auth code for access token                          │  │
│  │  - Fetches user info from provider                              │  │
│  │  - Creates/updates user in database                             │  │
│  │  - Generates Stormkit session token (JWT)                       │  │
│  │  - Redirects back to client app with token                      │  │
│  └──────────────────────────────────────────────────────────────────┘  │
└────────────────────────────────────┬────────────────────────────────────┘
                                     │
                                     │ 4. Redirect with session token
                                     │
                                     ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                          Client Application                              │
│  ┌──────────────────────────────────────────────────────────────────┐  │
│  │  - Stores session token                                          │  │
│  │  - Makes authenticated requests to app backend                   │  │
│  └──────────────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────────────┘
```

## Database Schema Relationships

```
┌───────────────────────┐
│      apps             │
│  ─────────────────    │
│  app_id (PK)         │
└───────┬───────────────┘
        │
        │ 1:N
        │
        ▼
┌───────────────────────────────┐
│   app_auth_providers          │
│  ──────────────────────────   │
│  provider_id (PK)             │
│  app_id (FK)                  │◄──────┐
│  provider_type (google, x)    │       │
│  client_id                    │       │
│  client_secret (encrypted)    │       │
│  scopes                       │       │ 1:N
│  enabled                      │       │
└───────────────────────────────┘       │
                                        │
        ┌───────────────────────────────┘
        │
        ▼
┌───────────────────────────────────┐
│   app_auth_users                  │
│  ──────────────────────────────   │
│  auth_user_id (PK)                │
│  app_id (FK)                      │
│  provider_id (FK)                 │◄──────┐
│  provider_user_id                 │       │
│  email                            │       │
│  display_name                     │       │ 1:N
│  avatar_url                       │       │
│  metadata (jsonb)                 │       │
│  last_sign_in_at                  │       │
└───────────────────────────────────┘       │
                                            │
        ┌───────────────────────────────────┘
        │
        ▼
┌───────────────────────────────────┐
│   app_auth_sessions               │
│  ──────────────────────────────   │
│  session_id (PK, UUID)            │
│  auth_user_id (FK)                │
│  access_token                     │
│  refresh_token                    │
│  expires_at                       │
│  user_agent                       │
│  ip_address                       │
│  created_at                       │
│  last_accessed_at                 │
└───────────────────────────────────┘
```

## API Endpoint Structure

```
┌─────────────────────────────────────────────────────────────────────┐
│                          Stormkit API Router                         │
└─────────────────────────────────────────────────────────────────────┘
                                  │
                    ┌─────────────┴──────────────┐
                    │                            │
                    ▼                            ▼
        ┌───────────────────────┐    ┌──────────────────────────┐
        │  Private/Dashboard    │    │  Public/Client APIs      │
        │  APIs                 │    │                          │
        └───────────────────────┘    └──────────────────────────┘
                    │                            │
        ┌───────────┴───────────┐    ┌──────────┴──────────────────┐
        │                       │    │                             │
        ▼                       ▼    ▼                             ▼
┌──────────────┐     ┌──────────────┐  ┌──────────────┐  ┌──────────────┐
│ Provider     │     │ User         │  │ Auth Login   │  │ Session      │
│ Management   │     │ Management   │  │ & Callback   │  │ Management   │
├──────────────┤     ├──────────────┤  ├──────────────┤  ├──────────────┤
│ • List       │     │ • List users │  │ • Login      │  │ • Get user   │
│ • Create     │     │ • View user  │  │ • Callback   │  │ • Refresh    │
│ • Update     │     │ • Delete user│  │ • Logout     │  │ • Validate   │
│ • Delete     │     │ • Search     │  │              │  │              │
└──────────────┘     └──────────────┘  └──────────────┘  └──────────────┘
```

## Data Flow: User Authentication

```
┌──────────┐                                         ┌──────────────┐
│  Client  │                                         │   Database   │
└────┬─────┘                                         └──────┬───────┘
     │                                                      │
     │ 1. GET /auth/google/login?redirect_uri=...          │
     ├────────────────────────────────────────┐            │
     │                                        │            │
     │                               ┌────────▼──────────┐ │
     │                               │  Auth Handler     │ │
     │                               │  - Load provider  │─┤─── Query provider config
     │                               │    config        │◄├───
     │                               │  - Generate state│ │
     │                               └────────┬──────────┘ │
     │                                        │            │
     │◄─── 2. Redirect to OAuth Provider ────┤            │
     │        with state token                │            │
     │                                        │            │
     │ 3. User authenticates at provider      │            │
     │                                        │            │
     │ 4. GET /callback?code=xxx&state=xxx    │            │
     ├────────────────────────────────────────┤            │
     │                                        │            │
     │                               ┌────────▼──────────┐ │
     │                               │ Callback Handler  │ │
     │                               │ - Validate state  │ │
     │                               │ - Exchange code   │ │
     │                               │ - Get user info   │ │
     │                               │ - Create session  │─┤─── Insert/update user
     │                               │                  │◄├───
     │                               │                  │─┤─── Create session
     │                               │                  │◄├───
     │                               └────────┬──────────┘ │
     │                                        │            │
     │◄─── 5. Redirect with token ────────────┤            │
     │                                        │            │
     │ 6. GET /auth/user                      │            │
     │    Authorization: Bearer <token>       │            │
     ├────────────────────────────────────────┤            │
     │                                        │            │
     │                               ┌────────▼──────────┐ │
     │                               │  User Handler     │ │
     │                               │  - Validate token │─┤─── Verify session
     │                               │  - Load user data │◄├───
     │                               └────────┬──────────┘ │
     │                                        │            │
     │◄─── 7. User data ──────────────────────┤            │
     │                                        │            │
     │                                                     │
```

## Security Flow

```
┌──────────────────────────────────────────────────────────────┐
│                    Security Measures                          │
└──────────────────────────────────────────────────────────────┘
                              │
        ┌─────────────────────┼─────────────────────┐
        │                     │                     │
        ▼                     ▼                     ▼
┌──────────────┐    ┌──────────────┐     ┌──────────────┐
│ Encryption   │    │ CSRF         │     │ Rate         │
│              │    │ Protection   │     │ Limiting     │
├──────────────┤    ├──────────────┤     ├──────────────┤
│ • Client     │    │ • State      │     │ • Login      │
│   secrets    │    │   token      │     │   attempts   │
│   (AES-256)  │    │   validation │     │ • API calls  │
│              │    │ • Nonce      │     │   per IP     │
│ • Session    │    │   checking   │     │ • Per user   │
│   tokens     │    │              │     │   limits     │
│   (JWT)      │    │              │     │              │
└──────────────┘    └──────────────┘     └──────────────┘
        │                     │                     │
        └─────────────────────┼─────────────────────┘
                              │
                              ▼
                    ┌──────────────────┐
                    │  Secure Storage  │
                    │  (PostgreSQL)    │
                    └──────────────────┘
```

## Integration Example

```javascript
// React Application Example
import React, { useEffect, useState } from 'react';

const STORMKIT_API = 'https://api.stormkit.io';
const APP_ID = '123';

function App() {
  const [user, setUser] = useState(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    // Check if user is already logged in
    const token = localStorage.getItem('auth_token');
    if (token) {
      fetchUser(token);
    } else {
      setLoading(false);
    }
  }, []);

  const fetchUser = async (token) => {
    try {
      const response = await fetch(
        `${STORMKIT_API}/public/auth/${APP_ID}/user`,
        {
          headers: {
            'Authorization': `Bearer ${token}`
          }
        }
      );
      if (response.ok) {
        const data = await response.json();
        setUser(data.user);
      } else {
        localStorage.removeItem('auth_token');
      }
    } catch (error) {
      console.error('Failed to fetch user:', error);
    } finally {
      setLoading(false);
    }
  };

  const loginWithGoogle = () => {
    const redirectUri = `${window.location.origin}/auth/callback`;
    const authUrl = `${STORMKIT_API}/public/auth/${APP_ID}/google/login?redirect_uri=${encodeURIComponent(redirectUri)}`;
    window.location.href = authUrl;
  };

  const logout = async () => {
    const token = localStorage.getItem('auth_token');
    await fetch(`${STORMKIT_API}/public/auth/${APP_ID}/logout`, {
      method: 'POST',
      headers: {
        'Authorization': `Bearer ${token}`
      }
    });
    localStorage.removeItem('auth_token');
    setUser(null);
  };

  if (loading) {
    return <div>Loading...</div>;
  }

  if (!user) {
    return (
      <div>
        <h1>Welcome!</h1>
        <button onClick={loginWithGoogle}>Login with Google</button>
      </div>
    );
  }

  return (
    <div>
      <h1>Welcome, {user.displayName}!</h1>
      <img src={user.avatarUrl} alt="Avatar" />
      <p>Email: {user.email}</p>
      <button onClick={logout}>Logout</button>
    </div>
  );
}

// Callback page component
function AuthCallback() {
  useEffect(() => {
    const params = new URLSearchParams(window.location.search);
    const token = params.get('token');
    const error = params.get('error');

    if (token) {
      localStorage.setItem('auth_token', token);
      window.location.href = '/';
    } else if (error) {
      console.error('Auth error:', error);
      window.location.href = '/?error=' + error;
    }
  }, []);

  return <div>Completing authentication...</div>;
}

export default App;
```

## Provider Configuration Flow

```
┌──────────────────────────────────────────────────────────────┐
│             Developer Setup in Stormkit Dashboard            │
└──────────────────────────────────────────────────────────────┘
                              │
                              │ 1. Navigate to App > Auth
                              │
                              ▼
                    ┌───────────────────┐
                    │  Auth Dashboard   │
                    │  • Provider cards │
                    │  • User list      │
                    │  • Docs panel     │
                    └─────────┬─────────┘
                              │
                              │ 2. Click "Configure Google"
                              │
                              ▼
                    ┌───────────────────┐
                    │  Provider Modal   │
                    │  ┌─────────────┐  │
                    │  │ Client ID   │  │
                    │  ├─────────────┤  │
                    │  │ Secret      │  │
                    │  ├─────────────┤  │
                    │  │ Scopes      │  │
                    │  ├─────────────┤  │
                    │  │ [Save]      │  │
                    │  └─────────────┘  │
                    └─────────┬─────────┘
                              │
                              │ 3. POST /app/:id/auth/providers
                              │
                              ▼
                    ┌───────────────────┐
                    │  Backend Handler  │
                    │  • Validate input │
                    │  • Encrypt secret │
                    │  • Store in DB    │
                    └─────────┬─────────┘
                              │
                              │ 4. Return success
                              │
                              ▼
                    ┌───────────────────┐
                    │  Provider Active  │
                    │  ┌─────────────┐  │
                    │  │ ✓ Google    │  │
                    │  │ Redirect URI│  │
                    │  │ Code Example│  │
                    │  └─────────────┘  │
                    └───────────────────┘
```
