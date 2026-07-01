import { useEffect, useState } from "react";
import XIcon from "@mui/icons-material/X";
import GoogleIcon from "@mui/icons-material/Google";
import EmailIcon from "@mui/icons-material/Email";
import LinkIcon from "@mui/icons-material/Link";
import MuiLink from "@mui/material/Link";
import api from "~/utils/api/Api";

// Duration unit table for the Session TTL field. Order matters for
// formatDuration: largest unit that divides cleanly wins.
const DURATION_UNITS: { suffix: string; minutes: number }[] = [
  { suffix: "y", minutes: 365 * 24 * 60 },
  { suffix: "mo", minutes: 30 * 24 * 60 },
  { suffix: "w", minutes: 7 * 24 * 60 },
  { suffix: "d", minutes: 24 * 60 },
  { suffix: "h", minutes: 60 },
  { suffix: "min", minutes: 1 },
];

// parseDuration turns "7d", "12h", "30min", "1w", "1mo", "1y" (case-
// insensitive, optional whitespace) into minutes. Returns null on garbage so
// the caller can surface a validation error.
export const parseDuration = (input: string): number | null => {
  const match = input
    .trim()
    .toLowerCase()
    .match(/^(\d+)\s*(min|mo|h|d|w|y)$/);

  if (!match) {
    return null;
  }

  const value = Number(match[1]);
  const unit = DURATION_UNITS.find(u => u.suffix === match[2]);

  if (!unit || !Number.isFinite(value)) {
    return null;
  }

  return value * unit.minutes;
};

// formatDuration renders minutes as the largest unit that divides evenly,
// falling back to plain minutes. 10080 → "1w", 360 → "6h", 1500 → "1500min".
export const formatDuration = (minutes: number): string => {
  if (!Number.isFinite(minutes) || minutes <= 0) {
    return "";
  }

  for (const unit of DURATION_UNITS) {
    if (minutes % unit.minutes === 0) {
      return `${minutes / unit.minutes}${unit.suffix}`;
    }
  }

  return `${minutes}min`;
};

export interface Field {
  name: string;
  label: string;
  value: string;
  required?: boolean;
  helperText?: string;
}

type AuthProviderID = "email" | "magiclink" | "google" | "x";

export interface AuthProvider {
  id: AuthProviderID;
  icon: typeof EmailIcon;
  name: string;
  drawerTitle: string;
  drawerDesc: string;
  fields?: Field[];
  enabled?: boolean;
  redirectUrl?: string;
  authUrl?: string;
  hasRedirectUrl?: boolean;
  hasAuthUrl?: boolean;
  steps?: React.ReactNode[];
}

const allProviders: AuthProvider[] = [
  {
    id: "email",
    icon: EmailIcon,
    name: "Email",
    drawerTitle: "Email Provider Settings",
    drawerDesc: "Allow authentication using email and password.",
    steps: [
      "Enable this provider to allow users to register and sign in with an email and password.",
      <>
        Use the{" "}
        <MuiLink
          href="https://www.stormkit.io/docs/features/authentication"
          target="_blank"
          rel="noreferrer noopener"
        >
          Stormkit Auth API
        </MuiLink>{" "}
        to register and authenticate users programmatically.
      </>,
      <>
        Send a <code>POST</code> request to{" "}
        <code>/_stormkit/auth/register</code> with a JSON body containing{" "}
        <code>email</code> and <code>password</code> fields. On success, a JWT
        token is returned.
      </>,
    ],
  },
  {
    id: "magiclink",
    icon: LinkIcon,
    name: "Magic Link",
    drawerTitle: "Magic Link Settings",
    drawerDesc: "Allow passwordless sign-in via a one-time link sent by email.",
    fields: [
      {
        name: "fromAddress",
        label: "From address",
        value: "",
        required: true,
        helperText:
          'Address used as the From header for magic-link emails (e.g. "Acme <noreply@acme.com>").',
      },
    ],
    steps: [
      "Enable this provider to allow users to sign in with a magic link sent to their email address.",
      <>
        Send a <code>GET</code> request to{" "}
        <code>/_stormkit/auth/magic?email=user@example.com</code>. A one-time
        sign-in link will be sent to that address. The endpoint returns{" "}
        <code>201</code> with no content.
      </>,
      <>
        The link redirects the user to{" "}
        <code>/_stormkit/auth/magic?token=&lt;token&gt;</code>, which exchanges
        the token for a session JWT stored in <code>localStorage</code>.
      </>,
    ],
  },
  {
    id: "google",
    icon: GoogleIcon,
    name: "Google",
    drawerTitle: "Google OAuth Settings",
    drawerDesc: "Sign in with Google OAuth 2.0.",
    hasRedirectUrl: true,
    hasAuthUrl: true,
    fields: [
      { name: "clientId", label: "Client ID", value: "" },
      { name: "clientSecret", label: "Client Secret", value: "" },
    ],
    steps: [
      <>
        Go to{" "}
        <MuiLink
          type="button"
          href="https://console.developers.google.com/apis/credentials"
          target="_blank"
          rel="noreferrer noopener"
        >
          https://console.developers.google.com/apis/credentials
        </MuiLink>
      </>,
      "Create a new OAuth 2.0 Client ID credential.",
      "Select 'Web application' as the Application type.",
      "Set the Authorized Redirect URI to the Redirect URL above.",
      "Copy the Client ID and Client Secret into the fields above.",
      "Save the settings and test the connection.",
    ],
  },
  {
    id: "x",
    icon: XIcon,
    name: "X / Twitter (OAuth 2.0)",
    drawerTitle: "X / Twitter OAuth Settings",
    drawerDesc: "Sign in with X OAuth 2.0.",
    hasRedirectUrl: true,
    hasAuthUrl: true,
    fields: [
      { name: "clientId", label: "Client ID", value: "" },
      { name: "clientSecret", label: "Client Secret", value: "" },
    ],
    steps: [
      <>
        Go to{" "}
        <MuiLink
          type="button"
          href="https://developer.x.com/en/portal/dashboard"
          target="_blank"
          rel="noreferrer noopener"
        >
          https://developer.x.com/en/portal/dashboard
        </MuiLink>{" "}
        and create a new project.
      </>,
      "Click 'Set up' under 'User authentication settings' section.",
      "Toggle 'Request email from users' on.",
      "Select 'Web App, Automated App or Bot'",
      "Set the Redirect URL as specified above.",
      "Copy the Client ID and Client Secret into the fields above.",
      "Save the settings and test the connection.",
    ],
  },
];

interface FetchProvidersParams {
  envId: string;
  refreshToken?: number;
}

interface ProviderData {
  status: boolean;
  clientId?: string;
  clientSecret?: string;
  redirectUrl?: string;
  [key: string]: any;
}

interface FetchProvidersResult {
  successUrl?: string;
  tokenTtl?: number;
  allowedOrigins?: string[];
  redirectUrl: string;
  authUrl: string;
  providers: {
    [key: string]: ProviderData;
  };
}

interface AuthConfig {
  sessionTtl?: number; // in minutes
  successUrl?: string;
  allowedOrigins?: string[];
}

interface SaveConfigParams {
  envId: string;
  successUrl: string;
  tokenTtl: number;
  status: boolean;
  allowedOrigins: string[];
}

export const saveConfig = ({
  envId,
  successUrl,
  tokenTtl,
  status,
  allowedOrigins,
}: SaveConfigParams): Promise<void> =>
  api.post("/skauth/config", {
    envId,
    successUrl,
    tokenTtl,
    status,
    allowedOrigins,
  });

export const useFetchProviders = ({
  envId,
  refreshToken,
}: FetchProvidersParams) => {
  const [error, setError] = useState<string>();
  const [loading, setLoading] = useState(true);
  const [providers, setProviders] = useState<AuthProvider[]>([]);
  const [config, setConfig] = useState<AuthConfig>({});

  useEffect(() => {
    setLoading(true);
    setError(undefined);

    api
      .fetch<FetchProvidersResult>(`/skauth/providers?envId=${envId}`)
      .then(
        ({
          providers,
          redirectUrl,
          authUrl,
          successUrl,
          tokenTtl: sessionTtl,
          allowedOrigins,
        }) => {
          const result: AuthProvider[] = [];

          allProviders
            .map(provider => ({
              ...provider,
              redirectUrl: provider.hasRedirectUrl ? redirectUrl : undefined,
              authUrl: provider.hasAuthUrl ? authUrl : undefined,
            }))
            .forEach(provider => {
              result.push({
                ...provider,
                fields: provider.fields?.map(field => ({
                  ...field,
                  value:
                    typeof providers[provider.id]?.[field.name] === "string"
                      ? (providers[provider.id]?.[field.name] as string)
                      : "",
                })),
                enabled: providers[provider.id]?.status === true,
              });
            });

          setProviders(result);
          setConfig({ successUrl, sessionTtl, allowedOrigins });
        },
      )
      .catch(() => {
        setError("Failed to fetch authentication providers");
      })
      .finally(() => {
        setLoading(false);
      });
  }, [envId, refreshToken]);

  return { providers, loading, error, config };
};

export interface AuthUser {
  id: string;
  firstName: string;
  lastName: string;
  email: string;
  avatar: string;
  createdAt: number;
  lastLoginAt: number;
}

interface UpdateAuthUserParams {
  envId: string;
  userId: string;
  email: string;
  firstName: string;
  lastName: string;
}

export const updateAuthUser = ({
  envId,
  userId,
  email,
  firstName,
  lastName,
}: UpdateAuthUserParams): Promise<AuthUser> =>
  api.put<AuthUser>(`/skauth/users/${userId}`, {
    envId,
    email,
    firstName,
    lastName,
  });

interface DeleteAuthUserParams {
  envId: string;
  userId: string;
}

export const deleteAuthUser = ({
  envId,
  userId,
}: DeleteAuthUserParams): Promise<void> =>
  api.delete(`/skauth/users/${userId}?envId=${envId}`);

interface FetchAuthUsersParams {
  envId: string;
  refreshToken?: number;
}

export const useFetchAuthUsers = ({
  envId,
  refreshToken,
}: FetchAuthUsersParams) => {
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string>();
  const [users, setUsers] = useState<AuthUser[]>([]);
  const [offset, setOffset] = useState(0);
  const [hasNextPage, setHasNextPage] = useState(false);

  const fetchPage = (from: number) => {
    setLoading(true);
    setError(undefined);

    api
      .fetch<{ results: AuthUser[]; hasNextPage: boolean }>(
        `/skauth/users?envId=${envId}&from=${from}`,
      )
      .then(({ results, hasNextPage }) => {
        setUsers(prev => (from === 0 ? results : [...prev, ...results]));
        setHasNextPage(hasNextPage);
        setOffset(from + results.length);
      })
      .catch(() => {
        setError("Something went wrong while fetching auth users.");
      })
      .finally(() => {
        setLoading(false);
      });
  };

  useEffect(() => {
    setOffset(0);
    setUsers([]);
    fetchPage(0);
  }, [envId, refreshToken]);

  const loadMore = () => {
    fetchPage(offset);
  };

  return { loading, error, users, hasNextPage, loadMore };
};
