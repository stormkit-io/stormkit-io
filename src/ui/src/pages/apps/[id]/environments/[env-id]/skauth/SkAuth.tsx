import { useContext, useState } from "react";
import Box from "@mui/material/Box";
import Button from "@mui/material/Button";
import Typography from "@mui/material/Typography";
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
import { AuthProvider, saveConfig, useFetchProviders } from "./actions";

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
        envId={env.id!}
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
  const { providers, loading, error, config } = useFetchProviders({
    envId: env.id!,
    refreshToken,
  });

  const hasSchema = !result.loading && !result.error && Boolean(result.schema);
  const title = "Authentication";
  const subtitle = "Enable authentication providers for this environment";

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

          saveConfig({
            envId: env.id!,
            successUrl: (data.get("successUrl") as string) || "",
            tokenTtl: Number(data.get("tokenTtl") || 0),
            status: true,
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
            }}
          />
        </Box>

        <Box sx={{ mb: 4 }}>
          <TextField
            label="Session TTL"
            name="tokenTtl"
            type="number"
            placeholder="10"
            fullWidth
            defaultValue={config?.sessionTtl || ""}
            variant="filled"
            autoComplete="off"
            helperText="Token lifetime in minutes"
            slotProps={{
              inputLabel: {
                shrink: true,
              },
            }}
          />
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
