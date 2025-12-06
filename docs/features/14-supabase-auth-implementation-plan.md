---
title: Supabase-like Auth Implementation Plan
description: High-level implementation plan for building a Supabase-like authentication system in Stormkit, focusing on OAuth provider management and user authentication flows.
---

# Supabase-like Auth Implementation Plan

## Overview

This document outlines the implementation plan for adding Supabase-like authentication capabilities to Stormkit. The feature will allow users to configure various OAuth providers (Google, X/Twitter, GitHub, etc.) through the Stormkit dashboard and provide authentication APIs for their applications.

## Problem Statement

Currently, Stormkit users who need authentication in their applications must:
1. Set up and manage their own authentication infrastructure
2. Handle OAuth flows manually in their application code
3. Manage user sessions and tokens independently
4. Context-switch between Stormkit and external authentication providers

This creates friction and slows down development for full-stack applications.

## Proposed Solution

Build a Supabase-like authentication system that:
1. Allows users to configure OAuth providers through the Stormkit dashboard
2. Provides backend APIs to handle OAuth flows automatically
3. Manages user sessions and tokens securely
4. Exposes simple authentication endpoints for client applications

## High-Level Architecture

### Phase 1: Auth Provider Management (Focus of this document)

#### 1. Database Schema

Add new tables to support app-level authentication:

```sql
-- Auth providers configured for an app
CREATE TABLE IF NOT EXISTS skitapi.app_auth_providers (
    provider_id SERIAL PRIMARY KEY,
    app_id BIGINT NOT NULL REFERENCES skitapi.apps(app_id) ON DELETE CASCADE,
    provider_type TEXT NOT NULL, -- 'google', 'x', 'github', 'facebook', etc.
    provider_name TEXT, -- Optional custom name (e.g., "Google Login", "Sign in with X")
    client_id TEXT NOT NULL,
    client_secret BYTEA NOT NULL, -- Encrypted
    redirect_uri TEXT,
    scopes TEXT[], -- OAuth scopes requested
    enabled BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITHOUT TIME ZONE DEFAULT (now() AT TIME ZONE 'UTC') NOT NULL,
    updated_at TIMESTAMP WITHOUT TIME ZONE DEFAULT (now() AT TIME ZONE 'UTC') NOT NULL,
    UNIQUE(app_id, provider_type)
);

-- App users who have authenticated through the auth system
CREATE TABLE IF NOT EXISTS skitapi.app_auth_users (
    auth_user_id BIGSERIAL PRIMARY KEY,
    app_id BIGINT NOT NULL REFERENCES skitapi.apps(app_id) ON DELETE CASCADE,
    provider_id BIGINT NOT NULL REFERENCES skitapi.app_auth_providers(provider_id) ON DELETE CASCADE,
    provider_user_id TEXT NOT NULL, -- User ID from the OAuth provider
    email TEXT NOT NULL,
    email_verified BOOLEAN DEFAULT false,
    display_name TEXT,
    avatar_url TEXT,
    metadata JSONB, -- Additional user data from provider
    last_sign_in_at TIMESTAMP WITHOUT TIME ZONE,
    created_at TIMESTAMP WITHOUT TIME ZONE DEFAULT (now() AT TIME ZONE 'UTC') NOT NULL,
    updated_at TIMESTAMP WITHOUT TIME ZONE DEFAULT (now() AT TIME ZONE 'UTC') NOT NULL,
    UNIQUE(app_id, provider_id, provider_user_id)
);

-- User sessions for authenticated app users
CREATE TABLE IF NOT EXISTS skitapi.app_auth_sessions (
    session_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    auth_user_id BIGINT NOT NULL REFERENCES skitapi.app_auth_users(auth_user_id) ON DELETE CASCADE,
    access_token TEXT NOT NULL,
    refresh_token TEXT,
    expires_at TIMESTAMP WITHOUT TIME ZONE NOT NULL,
    user_agent TEXT,
    ip_address TEXT,
    created_at TIMESTAMP WITHOUT TIME ZONE DEFAULT (now() AT TIME ZONE 'UTC') NOT NULL,
    last_accessed_at TIMESTAMP WITHOUT TIME ZONE DEFAULT (now() AT TIME ZONE 'UTC') NOT NULL
);

CREATE INDEX idx_app_auth_sessions_user_id ON skitapi.app_auth_sessions(auth_user_id);
CREATE INDEX idx_app_auth_sessions_expires_at ON skitapi.app_auth_sessions(expires_at);
CREATE INDEX idx_app_auth_users_app_id ON skitapi.app_auth_users(app_id);
CREATE INDEX idx_app_auth_users_email ON skitapi.app_auth_users(email);
```

#### 2. Backend API Structure

Create new Go packages in `src/ce/api/app/appauth/`:

```
src/ce/api/app/appauth/
├── appauth_model.go           # Data models for auth providers and users
├── appauth_store.go           # Database operations
├── appauth_statements.go      # SQL queries
├── providers/                 # OAuth provider implementations
│   ├── google.go
│   ├── twitter.go
│   ├── facebook.go
│   └── provider_interface.go
└── appauthhandlers/          # HTTP handlers
    ├── handler_providers_list.go
    ├── handler_providers_create.go
    ├── handler_providers_update.go
    ├── handler_providers_delete.go
    ├── handler_auth_login.go      # Initiate OAuth flow
    ├── handler_auth_callback.go   # Handle OAuth callback
    ├── handler_auth_user.go       # Get current user
    ├── handler_auth_logout.go     # Logout user
    └── services.go
```

#### 3. API Endpoints

##### Provider Management (Dashboard/Admin APIs)

```
# List all auth providers for an app
GET /app/:appId/auth/providers
Response: {
  "providers": [
    {
      "providerId": 1,
      "providerType": "google",
      "providerName": "Google Login",
      "clientId": "xxxxx",
      "redirectUri": "https://api.stormkit.io/app/123/auth/callback/google",
      "scopes": ["email", "profile"],
      "enabled": true,
      "createdAt": "2025-01-01T00:00:00Z"
    }
  ]
}

# Create a new auth provider
POST /app/:appId/auth/providers
Body: {
  "providerType": "google",
  "providerName": "Google Login",
  "clientId": "xxxxx",
  "clientSecret": "xxxxx",
  "scopes": ["email", "profile"]
}

# Update an auth provider
PATCH /app/:appId/auth/providers/:providerId
Body: {
  "providerName": "Updated Name",
  "enabled": true
}

# Delete an auth provider
DELETE /app/:appId/auth/providers/:providerId

# List authenticated users for an app
GET /app/:appId/auth/users
Response: {
  "users": [
    {
      "authUserId": 1,
      "email": "user@example.com",
      "displayName": "John Doe",
      "provider": "google",
      "lastSignInAt": "2025-01-01T00:00:00Z"
    }
  ]
}
```

##### Authentication APIs (Public/Client APIs)

These endpoints will be used by client applications:

```
# Initiate OAuth login flow
GET /public/auth/:appId/:providerType/login
Query params:
  - redirect_uri: Where to redirect after authentication
  - state: Optional CSRF token
Response: Redirects to OAuth provider

# OAuth callback handler
GET /public/auth/:appId/:providerType/callback
Query params:
  - code: OAuth authorization code
  - state: CSRF token
Response: Redirects to client app with session token or posts message to window.opener

# Get current authenticated user
GET /public/auth/:appId/user
Headers:
  - Authorization: Bearer <session_token>
Response: {
  "user": {
    "id": "uuid",
    "email": "user@example.com",
    "displayName": "John Doe",
    "avatarUrl": "https://...",
    "provider": "google",
    "metadata": {}
  }
}

# Refresh session token
POST /public/auth/:appId/refresh
Body: {
  "refreshToken": "xxxxx"
}
Response: {
  "accessToken": "xxxxx",
  "refreshToken": "xxxxx",
  "expiresAt": "2025-01-01T00:00:00Z"
}

# Logout
POST /public/auth/:appId/logout
Headers:
  - Authorization: Bearer <session_token>
Response: 204 No Content
```

#### 4. Frontend UI Components

Add new sections to the Stormkit dashboard:

```
src/ui/src/pages/apps/[id]/auth/
├── index.tsx                    # Main auth management page
├── _components/
│   ├── ProvidersList.tsx       # List of configured providers
│   ├── ProviderCard.tsx        # Individual provider display
│   ├── AddProviderModal.tsx    # Modal to add new provider
│   ├── EditProviderModal.tsx   # Modal to edit provider
│   ├── UsersList.tsx           # List of authenticated users
│   └── AuthDocsPanel.tsx       # Documentation panel with code examples
```

UI Flow:
1. Navigate to App > Auth (new tab in sidebar)
2. See list of available OAuth providers with "Configure" buttons
3. Click "Configure Google" → Modal opens with:
   - Client ID input
   - Client Secret input (encrypted/hidden)
   - Custom name input
   - Scopes selection
   - Enable/disable toggle
4. After configuration, see:
   - Provider card with status (enabled/disabled)
   - Auto-generated redirect URI
   - Code snippets for client integration
5. Separate tab for "Authenticated Users" showing:
   - Table of users who have signed in
   - Email, display name, provider, last sign-in time
   - Search/filter capabilities

#### 5. OAuth Provider Implementations

Each provider needs:

```go
type OAuthProvider interface {
    GetAuthURL(state, redirectURI string) string
    ExchangeCode(code string) (*oauth2.Token, error)
    GetUserInfo(token *oauth2.Token) (*AuthUser, error)
    GetProviderType() string
    GetDefaultScopes() []string
}

type AuthUser struct {
    ProviderUserID string
    Email          string
    EmailVerified  bool
    DisplayName    string
    AvatarURL      string
    Metadata       map[string]interface{}
}
```

Initial providers to support:
- Google OAuth 2.0
- X (formerly Twitter) OAuth 2.0
- GitHub OAuth (reuse existing implementation)
- Facebook OAuth 2.0

#### 6. Security Considerations

1. **Client Secrets**: Store encrypted in database using existing encryption utilities
2. **Session Tokens**: JWT tokens with short expiration (1 hour default)
3. **Refresh Tokens**: Longer-lived tokens for session renewal (30 days default)
4. **CSRF Protection**: Use state parameter in OAuth flows
5. **Rate Limiting**: Apply rate limits to auth endpoints
6. **CORS**: Properly configure CORS for auth endpoints
7. **Redirect URI Validation**: Validate redirect URIs against app's configured domains

#### 7. User Experience Flow

**Setting up Authentication:**

1. Developer navigates to their app in Stormkit dashboard
2. Clicks on "Auth" tab in sidebar (new)
3. Sees available OAuth providers
4. Clicks "Configure Google"
5. Enters Google OAuth credentials from Google Cloud Console
6. Optionally customizes provider name and scopes
7. Saves configuration
8. Gets redirect URI and code snippets for integration

**End-User Authentication:**

1. User clicks "Login with Google" button in the app
2. App redirects to: `https://api.stormkit.io/public/auth/123/google/login?redirect_uri=...`
3. Stormkit redirects user to Google OAuth consent screen
4. User approves and Google redirects back to Stormkit callback
5. Stormkit:
   - Exchanges code for token
   - Gets user info from Google
   - Creates/updates user record in database
   - Creates session
   - Redirects back to app with session token
6. App stores session token and uses it for authenticated requests

**Using Authentication in App:**

```javascript
// Client-side code example
const appId = '123';
const baseUrl = 'https://api.stormkit.io';

// Initiate login
function loginWithGoogle() {
  const redirectUri = `${window.location.origin}/auth/callback`;
  const authUrl = `${baseUrl}/public/auth/${appId}/google/login?redirect_uri=${encodeURIComponent(redirectUri)}`;
  window.location.href = authUrl;
}

// Handle callback
function handleCallback() {
  const params = new URLSearchParams(window.location.search);
  const token = params.get('token');
  if (token) {
    localStorage.setItem('auth_token', token);
    // Redirect to protected page
  }
}

// Get current user
async function getCurrentUser() {
  const token = localStorage.getItem('auth_token');
  const response = await fetch(`${baseUrl}/public/auth/${appId}/user`, {
    headers: {
      'Authorization': `Bearer ${token}`
    }
  });
  return response.json();
}

// Logout
async function logout() {
  const token = localStorage.getItem('auth_token');
  await fetch(`${baseUrl}/public/auth/${appId}/logout`, {
    method: 'POST',
    headers: {
      'Authorization': `Bearer ${token}`
    }
  });
  localStorage.removeItem('auth_token');
}
```

#### 8. Documentation Requirements

Create documentation pages:
- User guide for configuring OAuth providers
- API reference for authentication endpoints
- Integration examples for popular frameworks (React, Vue, Next.js)
- Security best practices
- Troubleshooting guide

## Implementation Phases

### Phase 1: Foundation (Current Focus)
- Database schema design ✓
- API endpoint design ✓
- High-level architecture ✓
- Provider interface design ✓

### Phase 2: Backend Implementation
- Create database migration
- Implement data models and store
- Implement Google OAuth provider
- Implement authentication handlers
- Add API endpoints to router
- Write backend tests

### Phase 3: Frontend Implementation
- Create Auth page in dashboard
- Implement provider configuration UI
- Implement user list view
- Add documentation panel with code examples
- Write frontend tests

### Phase 4: Additional Providers
- Implement X (formerly Twitter) OAuth
- Implement Facebook OAuth
- Add support for custom OAuth providers

### Phase 5: Advanced Features
- Email/password authentication
- Magic link authentication
- Multi-factor authentication
- Session management dashboard
- Webhook notifications for auth events

## Success Metrics

1. **User Adoption**: Number of apps using the auth feature
2. **Provider Usage**: Distribution of OAuth providers used
3. **Authentication Volume**: Number of auth requests per day
4. **Time to Setup**: Average time to configure first provider
5. **Error Rate**: Failed authentication attempts percentage

## Open Questions

1. Should we support custom OAuth providers where users can specify arbitrary OAuth endpoints?
2. How do we handle rate limiting for public auth endpoints to prevent abuse?
3. Should we provide a hosted user profile management UI or leave that to developers?
4. What's the token refresh strategy - automatic or manual?
5. Should we support SSO (Single Sign-On) across multiple Stormkit apps for the same user?
6. How do we handle provider deprecation (e.g., Twitter API changes)?

## Future Enhancements

1. **Social Login Widgets**: Pre-built UI components for login buttons
2. **User Management API**: CRUD operations for managing users programmatically
3. **Role-Based Access Control**: Assign roles to authenticated users
4. **Audit Logs**: Track authentication events
5. **Analytics**: Auth conversion rates, popular providers, etc.
6. **Passwordless Authentication**: Magic links, OTP
7. **Mobile SDK**: Native SDKs for iOS/Android
8. **Webhooks**: Notify apps of auth events (new user, login, etc.)

## Conclusion

This implementation plan provides a foundation for building a Supabase-like authentication system in Stormkit. The focus is on providing a simple, secure, and developer-friendly authentication solution that integrates seamlessly with the existing Stormkit platform.

The phased approach allows for iterative development and testing, starting with the most commonly used OAuth providers (Google) and expanding to others based on user demand.
