import { useContext, useEffect, useState } from "react";
import Alert from "@mui/material/Alert";
import Box from "@mui/material/Box";
import Button from "@mui/material/Button";
import Typography from "@mui/material/Typography";
import { Switch } from "~/components/Form";
import AddIcon from "@mui/icons-material/Add";
import OpenInNewIcon from "@mui/icons-material/OpenInNew";
import ChevronRightIcon from "@mui/icons-material/ChevronRight";
import IconButton from "@mui/material/IconButton";
import TextField from "@mui/material/TextField";
import { EnvironmentContext } from "~/pages/apps/[id]/environments/Environment.context";
import Card from "~/components/Card";
import CardHeader from "~/components/CardHeader";
import CardRow from "~/components/CardRow";
import EmptyPage from "~/components/EmptyPage";
import CardFooter from "~/components/CardFooter";
import { RootContext } from "~/pages/Root.context";
import Drawer from "./ProviderSettings";
import { useFetchSchema } from "../database/actions";
import {
  AuthProvider,
  formatDuration,
  parseDuration,
  saveConfig,
  useFetchProviders,
} from "./actions";

interface EmptyViewProps {
  isCloud?: boolean;
  env: Environment;
}

function EmptyView({ isCloud, env }: EmptyViewProps) {
  return (
    <EmptyPage>
      <Typography
        component="span"
        variant="h6"
        sx={{ mb: 4, display: "block" }}
      >
        {isCloud
          ? "The auth feature is currently only available for self-hosted installations."
          : "You need to attach a database to enable authentication providers."}
      </Typography>
      <Box component="span" sx={{ display: "block" }}>
        <Button
          href="https://www.stormkit.io/docs/features/database"
          variant="outlined"
          color="primary"
          target="_blank"
          rel="noreferrer noopener"
          endIcon={<OpenInNewIcon />}
        >
          Learn more
        </Button>
        {!isCloud && (
          <Button
            variant="contained"
            color="secondary"
            sx={{ ml: 2 }}
            href={`/apps/${env.appId}/environments/${env.id}/database`}
            startIcon={<AddIcon />}
          >
            Configure database
          </Button>
        )}
      </Box>
    </EmptyPage>
  );
}

interface ProvidersProps {
  providers: AuthProvider[];
  environment: Environment;
  setRefreshToken: (token: number) => void;
}

function Providers({
  providers,
  environment: env,
  setRefreshToken,
}: ProvidersProps) {
  const [drawerOpen, setDrawerOpen] = useState<AuthProvider>();

  return (
    <Card sx={{ mt: 2, width: "100%" }}>
      <CardHeader
        title="Providers"
        subtitle="Manage your authentication providers"
      />

      <Drawer
        isDrawerOpen={!!drawerOpen}
        setRefreshToken={setRefreshToken}
        onClose={() => {
          setDrawerOpen(undefined);
        }}
        provider={drawerOpen}
        environment={env}
      />

      {providers.map((p, i) => {
        const onClickHandler = () => {
          setDrawerOpen(p);
        };

        return (
          <CardRow
            key={p.id}
            sx={{
              cursor: "pointer",
              bgcolor: i % 2 ? "container.paper" : "transparent",
              p: 1,
              ":hover": {
                bgcolor: "rgba(0, 0, 0, 0.4)",
              },
            }}
            chipColor={p.enabled ? "success" : "info"}
            chipLabel={p.enabled ? "enabled" : "disabled"}
            chipSx={{ fontSize: 11, fontWeight: "normal" }}
            tabIndex={0}
            onClick={onClickHandler}
            actions={
              <Box sx={{ display: "flex", alignItems: "center", gap: 1 }}>
                <IconButton onClick={onClickHandler}>
                  <ChevronRightIcon />
                </IconButton>
              </Box>
            }
          >
            <Box
              sx={{
                display: "flex",
                gap: 2,
                alignItems: "center",
                justifyContent: "space-between",
              }}
            >
              <Box
                sx={{
                  display: "flex",
                  alignItems: "center",
                }}
              >
                {<p.icon />}
                <Typography component="span" sx={{ ml: 2 }}>
                  {p.name}
                </Typography>
              </Box>
            </Box>
          </CardRow>
        );
      })}
    </Card>
  );
}

export default function SkAuth() {
  const { details } = useContext(RootContext);
  const isCloud = details?.stormkit?.edition === "cloud";
  const { environment: env } = useContext(EnvironmentContext);
  const result = useFetchSchema({ envId: env.id!, isCloud });
  const [refreshToken, setRefreshToken] = useState<number>();
  const [success, setSuccess] = useState<string>();
  const [saving, setSaving] = useState(false);
  const [saveError, setSaveError] = useState<string>();
  const originsHintKey = `skauth_origins_hint_dismissed_${env.id}`;
  const [originsHintDismissed, setOriginsHintDismissed] = useState(
    () => localStorage.getItem(originsHintKey) === "1",
  );

  const dismissOriginsHint = () => {
    localStorage.setItem(originsHintKey, "1");
    setOriginsHintDismissed(true);
  };
  const { providers, loading, error, config } = useFetchProviders({
    envId: env.id!,
    refreshToken,
  });

  // The OAuth toggles are controlled Switches (not part of FormData), so they
  // need their own state, synced from the config once it loads.
  const [oauthServerEnabled, setOauthServerEnabled] = useState(false);
  const [oauthAllowLoopback, setOauthAllowLoopback] = useState(false);

  useEffect(() => {
    setOauthServerEnabled(Boolean(config?.oauthServerEnabled));
    setOauthAllowLoopback(Boolean(config?.oauthAllowLoopback));
  }, [config?.oauthServerEnabled, config?.oauthAllowLoopback]);

  const hasSchema = !result.loading && !result.error && Boolean(result.schema);
  const title = "Authentication";
  const subtitle = "Enable authentication providers for this environment";

  // Guide decoupled (separate frontend domain / native) setups: an enabled
  // provider with no allow-list silently returns users to this environment.
  const showOriginsHint =
    !originsHintDismissed &&
    providers.some(p => p.enabled) &&
    (config?.allowedOrigins || []).length === 0;

  if (!hasSchema && !result.loading) {
    return (
      <Card sx={{ width: "100%" }}>
        <CardHeader title={title} subtitle={subtitle} />
        <EmptyView isCloud={isCloud} env={env} />
      </Card>
    );
  }

  return (
    <>
      <Card
        success={success}
        successTitle={false}
        onSuccessClose={() => setSuccess(undefined)}
        error={result.error || error || saveError}
        loading={result.loading || loading}
        component="form"
        sx={{ width: "100%" }}
        onSubmit={e => {
          e.preventDefault();
          const form = e.target;

          if (!(form instanceof HTMLFormElement)) {
            throw new TypeError("Expected form submit target");
          }

          const data = new FormData(form);

          setSaving(true);
          setSaveError(undefined);

          const allowedOrigins = ((data.get("allowedOrigins") as string) || "")
            .split("\n")
            .map(line => line.trim())
            .filter(Boolean);

          const ttlInput = ((data.get("tokenTtl") as string) || "").trim();
          const tokenTtl = ttlInput ? parseDuration(ttlInput) : 0;

          if (ttlInput && tokenTtl === null) {
            setSaveError(
              "Session TTL must be a duration like 7d, 12h, 30min, or 1w.",
            );
            setSaving(false);
            return;
          }

          saveConfig({
            envId: env.id!,
            successUrl: (data.get("successUrl") as string) || "",
            tokenTtl: tokenTtl || 0,
            status: true,
            allowedOrigins,
            oauthServerEnabled,
            oauthResourcePath:
              ((data.get("oauthResourcePath") as string) || "").trim(),
            oauthAllowLoopback,
          })
            .then(() => {
              setSuccess("Settings saved successfully");
            })
            .catch(() => {
              setSaveError("Failed to save settings");
            })
            .finally(() => {
              setSaving(false);
            });
        }}
      >
        <CardHeader title={title} subtitle={subtitle} />

        <Box sx={{ mb: 4 }}>
          <TextField
            label="Success callback URL"
            name="successUrl"
            placeholder="/auth/success"
            fullWidth
            defaultValue={config?.successUrl || ""}
            variant="filled"
            autoComplete="off"
            helperText="Relative URL to redirect to after successful authentication. At this URL, browser code can read the skauth item from localStorage."
            slotProps={{
              inputLabel: {
                shrink: true,
              },
              htmlInput: {
                "data-lpignore": "true",
                "data-1p-ignore": "true",
                "data-bwignore": "true",
                "data-form-type": "other",
              },
            }}
          />
        </Box>

        <Box sx={{ mb: 4 }}>
          <TextField
            label="Session TTL"
            name="tokenTtl"
            placeholder="7d"
            fullWidth
            defaultValue={
              config?.sessionTtl ? formatDuration(config.sessionTtl) : ""
            }
            variant="filled"
            autoComplete="off"
            helperText="Token lifetime. Accepts e.g. 30min, 12h, 7d, 1w, 1mo."
            slotProps={{
              inputLabel: {
                shrink: true,
              },
            }}
          />
        </Box>

        <Box sx={{ mb: 4 }}>
          <TextField
            label="Allowed origins"
            name="allowedOrigins"
            placeholder={"https://app.example.com\nhttps://dev.example.com"}
            fullWidth
            multiline
            minRows={3}
            defaultValue={(config?.allowedOrigins || []).join("\n")}
            variant="filled"
            autoComplete="off"
            helperText="Optional. One origin per line (scheme + host, no path). Origins allowed for cross-origin sign-in. Leave empty for single-host setups."
            slotProps={{
              inputLabel: {
                shrink: true,
              },
            }}
          />
          {showOriginsHint && (
            <Alert severity="warning" sx={{ mt: 2 }} onClose={dismissOriginsHint}>
              A provider is enabled but no allowed origins are set. If your
              frontend runs on a separate domain (or a native app), add its
              origin — otherwise sign-in returns users here instead of your app.
            </Alert>
          )}
        </Box>

        <Box sx={{ mb: 4 }}>
          <Switch
            name="oauthServerEnabled"
            checked={oauthServerEnabled}
            setChecked={setOauthServerEnabled}
            label="Enable OAuth server (MCP connectors)"
            description="Turns this environment into an OAuth 2.1 authorization server so AI clients like ChatGPT and Claude can connect to it as an MCP server, signing in your app's own users. The Claude and ChatGPT connector origins are trusted automatically — no manual origin setup needed."
          />
          {oauthServerEnabled && (
            <>
              <Box sx={{ mt: 3 }}>
                <TextField
                  label="MCP server path"
                  name="oauthResourcePath"
                  placeholder="/mcp"
                  fullWidth
                  defaultValue={config?.oauthResourcePath || ""}
                  variant="filled"
                  autoComplete="off"
                  helperText="The path this environment serves its MCP server on. Connectors require the resource identifier to match the URL you enter (path included), so set this to the connector URL's path — e.g. /mcp for https://your-app.com/mcp."
                  slotProps={{
                    inputLabel: {
                      shrink: true,
                    },
                  }}
                />
              </Box>
              <Box sx={{ mt: 3 }}>
                <Switch
                  name="oauthAllowLoopback"
                  checked={oauthAllowLoopback}
                  setChecked={setOauthAllowLoopback}
                  label="Allow native / CLI clients"
                  description="Permit RFC 8252 loopback redirects (http://127.0.0.1 and http://localhost on any port) so native tools like Claude Code can connect. Leave off if you only use browser-based connectors."
                />
              </Box>
              <Alert severity="info" sx={{ mt: 2 }}>
                <Typography sx={{ mb: 1 }}>
                  Discovery documents, served on this environment's domain:
                </Typography>
                <Box component="code" sx={{ display: "block", fontSize: 12 }}>
                  /.well-known/oauth-authorization-server
                </Box>
                <Box component="code" sx={{ display: "block", fontSize: 12 }}>
                  /.well-known/oauth-protected-resource
                  {config?.oauthResourcePath || ""}
                </Box>
                <Typography sx={{ mt: 2, mb: 1 }}>
                  Connectors self-register via Dynamic Client Registration (RFC
                  7591); no manual client setup is needed:
                </Typography>
                <Box component="code" sx={{ display: "block", fontSize: 12 }}>
                  POST /_stormkit/oauth/register
                </Box>
              </Alert>
            </>
          )}
        </Box>

        <CardFooter>
          <Button
            type="submit"
            variant="contained"
            color="secondary"
            disabled={result.loading || loading || saving}
          >
            Save
          </Button>
        </CardFooter>
      </Card>
      {hasSchema && (
        <Providers
          providers={providers}
          environment={env}
          setRefreshToken={setRefreshToken}
        />
      )}
    </>
  );
}
