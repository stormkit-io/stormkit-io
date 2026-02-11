import { useEffect, useState } from "react";
import XIcon from "@mui/icons-material/X";
import GoogleIcon from "@mui/icons-material/Google";
import EmailIcon from "@mui/icons-material/Email";
import Link from "@mui/material/Link";
import api from "~/utils/api/Api";

export interface Field {
  name: string;
  label: string;
  value: string;
  helperText?: string;
}

type AuthProviderID = "email" | "google" | "x";

export interface AuthProvider {
  id: AuthProviderID;
  icon: typeof EmailIcon;
  name: string;
  drawerTitle: string;
  drawerDesc: string;
  fields?: Field[];
  enabled?: boolean;
  redirectUrl?: string;
  hasRedirectUrl?: boolean;
  steps?: React.ReactNode[];
}

const allProviders: AuthProvider[] = [
  // {
  //   id: "email",
  //   icon: EmailIcon,
  //   name: "Email",
  //   drawerTitle: "Email Provider Settings",
  //   drawerDesc: "Allow authentication using email/password and magic links.",
  //   fields: [
  //     { name: "clientId", label: "Client ID", value: "" },
  //     { name: "clientSecret", label: "Client Secret", value: "" },
  //   ],
  // },
  {
    id: "google",
    icon: GoogleIcon,
    name: "Google",
    drawerTitle: "Google OAuth Settings",
    drawerDesc: "Sign in with Google OAuth 2.0.",
    hasRedirectUrl: true,
    fields: [
      { name: "clientId", label: "Client ID", value: "" },
      { name: "clientSecret", label: "Client Secret", value: "" },
    ],
    steps: [
      <>
        Go to{" "}
        <Link
          type="button"
          href="https://console.developers.google.com/apis/credentials"
          target="_blank"
          rel="noreferrer noopener"
        >
          https://console.developers.google.com/apis/credentials
        </Link>
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
    fields: [
      { name: "clientId", label: "Client ID", value: "" },
      { name: "clientSecret", label: "Client Secret", value: "" },
    ],
    steps: [
      <>
        Go to{" "}
        <Link
          type="button"
          href="https://developer.x.com/en/portal/dashboard"
          target="_blank"
          rel="noreferrer noopener"
        >
          https://developer.x.com/en/portal/dashboard
        </Link>{" "}
        and create a new project.
      </>,
      "Enable OAuth 2.0 and set the callback URL to the Redirect URL above.",
      "Copy the Client ID and Client Secret into the fields above.",
      "Save the settings and test the connection.",
    ],
  },
];

interface FetchProvidersParams {
  envId: string;
}

interface ProviderData {
  status: boolean;
  clientId?: string;
  clientSecret?: string;
  redirectUrl?: string;
  [key: string]: any;
}

interface FetchProvidersResult {
  redirectUrl: string;
  providers: {
    [key: string]: ProviderData;
  };
}

export const useFetchProviders = ({ envId }: FetchProvidersParams) => {
  const [error, setError] = useState<string>();
  const [loading, setLoading] = useState(true);
  const [providers, setProviders] = useState<AuthProvider[]>([]);

  useEffect(() => {
    setLoading(true);
    setError(undefined);

    api
      .fetch<FetchProvidersResult>(`/skauth/providers?envId=${envId}`)
      .then(({ providers, redirectUrl }) => {
        const result: AuthProvider[] = [];

        allProviders
          .map(provider => ({
            ...provider,
            redirectUrl: provider.hasRedirectUrl ? redirectUrl : undefined,
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
      })
      .catch(e => {
        setError("Failed to fetch authentication providers");
      })
      .finally(() => {
        setLoading(false);
      });
  }, [envId]);

  return { providers, loading, error };
};
