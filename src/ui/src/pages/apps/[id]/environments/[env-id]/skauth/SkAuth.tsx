import { useContext, useState } from "react";
import Box from "@mui/material/Box";
import Button from "@mui/material/Button";
import Typography from "@mui/material/Typography";
import AddIcon from "@mui/icons-material/Add";
import OpenInNewIcon from "@mui/icons-material/OpenInNew";
import ChevronRightIcon from "@mui/icons-material/ChevronRight";
import IconButton from "@mui/material/IconButton";
import { EnvironmentContext } from "~/pages/apps/[id]/environments/Environment.context";
import Card from "~/components/Card";
import CardHeader from "~/components/CardHeader";
import CardRow from "~/components/CardRow";
import EmptyPage from "~/components/EmptyPage";
import CardFooter from "~/components/CardFooter";
import { RootContext } from "~/pages/Root.context";
import Drawer from "./ProviderSettings";
import { useFetchSchema } from "../database/actions";
import { useFetchProviders } from "./actions";

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

export default function SkAuth() {
  const { details } = useContext(RootContext);
  const isCloud = details?.stormkit?.edition === "cloud";
  const { environment: env } = useContext(EnvironmentContext);
  const result = useFetchSchema({ envId: env.id!, isCloud });
  const [refreshToken, setRefreshToken] = useState<number>();
  const [success, setSuccess] = useState<string>();
  const [drawerOpen, setDrawerOpen] = useState<string>("");
  const { providers, loading, error } = useFetchProviders({
    envId: env.id!,
    refreshToken,
  });

  const hasSchema = !result.loading && !result.error && Boolean(result.schema);

  return (
    <Card
      success={success}
      successTitle={false}
      onSuccessClose={() => setSuccess(undefined)}
      error={result.error || error}
      loading={result.loading || loading}
      sx={{ width: "100%" }}
    >
      <CardHeader
        title="Authentication"
        subtitle="Enable authentication providers for this environment"
      />

      {!hasSchema ? (
        <EmptyView isCloud={isCloud} env={env} />
      ) : (
        <>
          {providers.map((p, i) => {
            const onClickHandler = () => {
              setDrawerOpen(p.id);
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
                chipLabel={p.enabled ? "Enabled" : "Disabled"}
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
                  tabIndex={0}
                  onClick={onClickHandler}
                >
                  <Box
                    sx={{
                      display: "flex",
                      alignItems: "center",
                      cursor: "pointer",
                    }}
                    onClick={onClickHandler}
                  >
                    {<p.icon />}
                    <Typography component="span" sx={{ ml: 2 }}>
                      {p.name}
                    </Typography>
                  </Box>
                </Box>
                <Drawer
                  isDrawerOpen={drawerOpen === p.id}
                  setRefreshToken={setRefreshToken}
                  onClose={() => {
                    setDrawerOpen("");
                  }}
                  provider={p}
                  envId={env.id!}
                />
              </CardRow>
            );
          })}

          <CardFooter sx={{ mt: 4 }} />
        </>
      )}
    </Card>
  );
}
