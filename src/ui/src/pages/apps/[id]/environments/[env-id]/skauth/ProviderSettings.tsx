import { useEffect, useMemo, useState } from "react";
import Box from "@mui/material/Box";
import Typography from "@mui/material/Typography";
import Drawer from "@mui/material/Drawer";
import TextField from "@mui/material/TextField";
import MenuItem from "@mui/material/MenuItem";
import Button from "@mui/material/Button";
import { Switch } from "~/components/Form";
import Card from "~/components/Card";
import CardHeader from "~/components/CardHeader";
import CardFooter from "~/components/CardFooter";
import CopyBox from "~/components/CopyBox";
import Api from "~/utils/api/Api";
import { useFetchDomains } from "~/shared/domains/actions";
import type { AuthProvider } from "./actions";

interface Props {
  environment: Environment;
  isDrawerOpen: boolean;
  provider?: AuthProvider;
  onClose: () => void;
  setRefreshToken: (value: number) => void;
}

export default function ProviderSettings({
  isDrawerOpen,
  provider,
  environment,
  onClose,
  setRefreshToken,
}: Props) {
  const envId = environment.id!;
  const [isEnabled, setIsEnabled] = useState(!!provider?.enabled);
  const [error, setError] = useState<string>();
  const [loading, setLoading] = useState(false);

  const [selectedOrigin, setSelectedOrigin] = useState("");

  // Verified custom domains plus the default preview domain are the hosts the
  // app is actually served from. The OAuth callback URL must be registered with
  // the provider for whichever domain users sign in on, so we let the customer
  // pick the domain and copy the matching callback URL.
  const { domains } = useFetchDomains({
    appId: environment.appId,
    envId,
    search: "",
    verified: true,
    enabled: isDrawerOpen && !!provider?.hasRedirectUrl,
  });

  const callbackOrigins = useMemo(() => {
    // `preview` already carries a scheme; custom domains are bare hosts. Coerce
    // both to a normalized origin so the callback URL is always well-formed.
    const toOrigin = (host?: string): string => {
      const trimmed = host?.trim();

      if (!trimmed) {
        return "";
      }

      const withScheme = /^https?:\/\//.test(trimmed)
        ? trimmed
        : `https://${trimmed}`;

      try {
        return new URL(withScheme).origin;
      } catch {
        return "";
      }
    };

    const origins = [environment.preview, ...domains.map(d => d.domainName)]
      .map(toOrigin)
      .filter(Boolean);

    return Array.from(new Set(origins));
  }, [environment.preview, domains]);

  // The preview origin is always the first candidate; flag it so it can be
  // labelled in the dropdown and told apart from verified custom domains.
  const previewOrigin = callbackOrigins[0];

  // Default the dropdown to the first known domain, keeping the current choice
  // if it survives a domains refresh.
  useEffect(() => {
    setSelectedOrigin(prev =>
      callbackOrigins.includes(prev) ? prev : callbackOrigins[0] || "",
    );
  }, [callbackOrigins]);

  const callbackOrigin = selectedOrigin || "https://<your-host>";

  useEffect(() => {
    setIsEnabled(!!provider?.enabled);
  }, [provider]);

  return (
    <Drawer
      anchor="right"
      open={isDrawerOpen}
      onClose={onClose}
      sx={{ zIndex: 1600 }}
    >
      <Card
        component="form"
        error={error}
        sx={{ minHeight: "100vh", minWidth: "600px", maxWidth: "800px" }}
        onSubmit={e => {
          e.preventDefault();
          const form = e.target as HTMLFormElement;
          const data = Object.fromEntries(
            new FormData(form).entries(),
          ) as Record<string, string>;

          setLoading(true);
          setError(undefined);

          Api.post(`/skauth`, {
            envId,
            providerName: provider?.id,
            clientId: data.clientId,
            clientSecret: data.clientSecret,
            fromAddress: data.fromAddress,
            status: isEnabled,
          })
            .then(() => {
              setRefreshToken(Date.now());
              onClose();
            })
            .catch(res => {
              if (res.status === 400) {
                res.json().then(({ error }: { error: string }) => {
                  setError(error);
                });
              } else {
                setError(
                  "Something went wrong while saving provider settings.",
                );
              }
            })
            .finally(() => {
              setLoading(false);
            });
        }}
      >
        <CardHeader
          title={provider?.drawerTitle}
          subtitle={provider?.drawerDesc}
        />
        <Box>
          {provider?.fields?.map(field => (
            <TextField
              variant="filled"
              autoComplete="off"
              key={field.name}
              name={field.name}
              label={field.label}
              defaultValue={field.value}
              helperText={field.helperText}
              required={field.required}
              fullWidth
              sx={{ mb: 2 }}
            />
          ))}

          <Switch
            checked={isEnabled}
            setChecked={setIsEnabled}
            name="status"
            label="Enable provider"
            description="Allow or disallow sign-in with this provider. Disabling will not delete existing users."
          />
        </Box>

        {provider?.hasRedirectUrl && (
          <Box
            sx={{
              mt: 2,
              border: "1px solid",
              p: 2,
              borderRadius: 1,
              bgcolor: "container.paper",
              borderColor: "container.border",
            }}
          >
            {callbackOrigins.length > 1 && (
              <TextField
                select
                fullWidth
                label="Domain"
                variant="filled"
                value={selectedOrigin}
                onChange={e => setSelectedOrigin(e.target.value)}
                helperText="Select one of these domains as the callback URL. To support multiple domains, create a separate app for each one."
                sx={{ mb: 2 }}
              >
                {callbackOrigins.map(origin => (
                  <MenuItem key={origin} value={origin}>
                    {origin}
                    {origin === previewOrigin ? " (preview)" : ""}
                  </MenuItem>
                ))}
              </TextField>
            )}

            <CopyBox
              fullWidth
              label="Callback URL"
              variant="filled"
              value={`${callbackOrigin}${provider.redirectUrl}`}
              helperText="Set this as the Redirect URL in your OAuth provider settings."
            />
          </Box>
        )}

        <Box
          component="ul"
          sx={{
            my: 2,
            border: "1px solid",
            p: 2,
            borderRadius: 1,
            borderColor: "container.border",
          }}
        >
          {provider?.steps?.map((step, index) => (
            <Typography
              component="li"
              key={index}
              sx={{ mb: 1, "&:last-child": { mb: 0 } }}
            >
              {index + 1}. {step}
            </Typography>
          ))}
        </Box>

        {provider?.hasAuthUrl && (
          <Box
            sx={{
              my: 2,
              border: "1px solid",
              p: 2,
              borderRadius: 1,
              bgcolor: "container.paper",
              borderColor: "container.border",
            }}
          >
            <CopyBox
              fullWidth
              label="Authorization URL"
              variant="filled"
              value={`${provider.authUrl}/${provider.id}`}
              helperText="Path on your app's own domain — redirect users here to start sign-in (e.g. https://app.example.com/_stormkit/auth/google). Optionally append ?redirect=<origin> to control where users return afterwards; otherwise the Origin/Referer is used."
            />
          </Box>
        )}
        <CardFooter>
          <Button onClick={onClose} sx={{ mr: 2 }} variant="outlined">
            Cancel
          </Button>
          <Button
            variant="contained"
            color="secondary"
            type="submit"
            loading={loading}
          >
            Save
          </Button>
        </CardFooter>
      </Card>
    </Drawer>
  );
}
