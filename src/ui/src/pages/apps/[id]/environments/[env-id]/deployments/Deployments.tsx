import { useContext, useState } from "react";
import Box from "@mui/material/Box";
import EmptyPage from "~/components/EmptyPage";
import Card from "~/components/Card";
import CardHeader from "~/components/CardHeader";
import Deployment from "~/shared/deployments/DeploymentRow";
import { AppContext } from "~/pages/apps/[id]/App.context";
import { EnvironmentContext } from "~/pages/apps/[id]/environments/Environment.context";
import { useFetchDeployments } from "./actions";

export default function Deployments() {
  const { app } = useContext(AppContext);
  const { environment } = useContext(EnvironmentContext);
  const [refreshToken, setRefreshToken] = useState(0);
  const { deployments, loading, error } = useFetchDeployments({
    from: 0,
    app,
    refreshToken,
    filters: { envId: environment?.id },
  });

  return (
    <Card
      maxWidth="lg"
      sx={{ width: "100%" }}
      loading={loading}
      error={error}
      contentPadding={false}
    >
      <CardHeader
        title="Deployments"
        subtitle="View all of your deployments within this environment."
      />
      <Box>
        {deployments.map(deployment => (
          <Deployment
            key={deployment.id}
            deployment={deployment}
            setRefreshToken={setRefreshToken}
          />
        ))}
      </Box>
      {!loading && !error && !deployments.length && (
        <EmptyPage>
          It's quite empty in here. <br />
          Click the Deploy button to start your first deployment.
        </EmptyPage>
      )}
    </Card>
  );
}
