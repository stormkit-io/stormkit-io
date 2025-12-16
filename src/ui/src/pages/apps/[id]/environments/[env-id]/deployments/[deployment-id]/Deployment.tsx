import { useState } from "react";
import { useParams } from "react-router";
import Button from "@mui/material/Button";
import Card from "~/components/Card";
import CardHeader from "~/components/CardHeader";
import CardFooter from "~/components/CardFooter";
import Error404 from "~/components/Errors/Error404";
import Spinner from "~/components/Spinner";
import { useFetchDeployment, useWithPageRefresh } from "../actions";
import DeploymentRow from "~/shared/deployments/DeploymentRow";
import DeploymentLogs from "./DeploymentLogs";

export default function Deployment() {
  const { deploymentId } = useParams();
  const [refreshToken, setRefreshToken] = useState<number>();
  const { deployment, error, loading } = useFetchDeployment({
    deploymentId,
    refreshToken,
  });

  useWithPageRefresh({ deployment, setRefreshToken });

  const isRunningLogs =
    deployment?.status === "running" && !deployment.stoppedAt;
  const isRunningStatusChecks =
    deployment?.status === "running" && Boolean(deployment.stoppedAt);

  const snapshot = deployment?.snapshot;
  const hasStatusChecks = Boolean(snapshot?.build?.statusChecks?.length);
  const showPreviewButton =
    deployment?.status === "success" || (hasStatusChecks && !isRunningLogs);

  const showEmptyPackageWarning =
    deployment &&
    !isRunningLogs &&
    !deployment.stoppedAt &&
    !deployment.uploadResult;

  if (!deployment && !loading && !error) {
    return <Error404>Deployment is not found.</Error404>;
  }

  return (
    <>
      <Card
        sx={{ width: "100%" }}
        error={error}
        loading={loading}
        contentPadding={false}
        info={
          showEmptyPackageWarning ? (
            <>
              Deployment package is empty. Make sure that the build folder is
              specified properly.
            </>
          ) : (
            ""
          )
        }
      >
        <CardHeader
          sx={{
            py: 2,
            px: 0,
            pb: 0,
            mb: 2,
          }}
        >
          {deployment && (
            <DeploymentRow
              deployment={deployment}
              setRefreshToken={setRefreshToken}
              viewDetails={false}
              exactTime
              sx={{ borderTop: "none !important" }}
            />
          )}
        </CardHeader>
        <DeploymentLogs
          logs={deployment?.logs || []}
          isRunning={isRunningLogs}
        />
        <CardFooter sx={{ display: "flex", justifyContent: "center" }}>
          {isRunningLogs && <Spinner primary />}
          {showPreviewButton && (
            <Button
              href={deployment!.previewUrl}
              variant="contained"
              color="secondary"
            >
              Preview
            </Button>
          )}
        </CardFooter>
      </Card>
      <Card
        sx={{
          width: "100%",
          mt: 2,
          display:
            hasStatusChecks &&
            deployment?.stoppedAt &&
            (deployment.statusChecks?.length || 0) > 0
              ? "block"
              : "none",
        }}
        contentPadding={false}
      >
        <CardHeader title="Status checks" />
        <DeploymentLogs
          logs={deployment?.statusChecks || []}
          isRunning={isRunningStatusChecks}
        />
      </Card>
    </>
  );
}
