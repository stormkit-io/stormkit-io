import type { SxProps } from "@mui/material";
import { useState } from "react";
import { useNavigate } from "react-router";
import Box from "@mui/material/Box";
import Typography from "@mui/material/Typography";
import CircularProgress from "@mui/material/CircularProgress";
import LensIcon from "@mui/icons-material/Lens";
import CardRow from "~/components/CardRow";
import ConfirmModal from "~/components/ConfirmModal";
import { timeSince } from "~/utils/helpers/date";
import CommitInfo from "./CommitInfo";
import PublishModal from "./PublishModal";
import ManifestModal from "./ManifestModal";
import { deleteForever, stopDeployment } from "./actions";
import { useWithMenuItems, restartDeployment } from "./actions";
import { formattedDate } from "~/utils/helpers/deployments";

interface Props {
  deployment: DeploymentV2;
  showProject?: boolean;
  exactTime?: boolean;
  viewDetails?: boolean;
  setRefreshToken: (v: number) => void;
  sx?: SxProps;
}

const iconProps = {
  fontSize: 12,
  position: "relative",
  top: 3.5,
  mr: 2,
};

export default function DeploymentRow({
  deployment,
  setRefreshToken,
  showProject,
  viewDetails = true,
  exactTime,
  sx,
}: Props) {
  const isRunning = deployment.status === "running";
  const isSuccess = deployment.status === "success";
  const navigate = useNavigate();
  const [showStopModal, setShowStopModal] = useState<boolean>();
  const [showManifest, setShowManifest] = useState<boolean>();
  const [showDeleteModal, setShowDeleteModal] = useState<boolean>();
  const [showPublishModal, setShowPublishModal] = useState<boolean>();
  const menuItems = useWithMenuItems({
    deployment,
    omittedItems: viewDetails ? [] : ["view-details"],
    onManifestClick: () => setShowManifest(true),
    onStateChangeClick: () =>
      isRunning ? setShowStopModal(true) : setShowDeleteModal(true),
    onPublishClick: () => setShowPublishModal(true),
    onRestartClick: () => {
      restartDeployment({
        appId: deployment.appId,
        deploymentId: deployment.id,
        envId: deployment.envId,
      }).then(() => {
        navigate(
          `/apps/${deployment.appId}/environments/${deployment.envId}/deployments/${deployment.id}`,
        );
        setRefreshToken(Date.now());
      });
    },
  });

  return (
    <CardRow
      menuLabel={`Deployment ${deployment.id} menu`}
      menuItems={menuItems}
      actions={
        <Typography
          sx={{
            fontSize: 11,
            color: "text.secondary",
          }}
        >
          {exactTime
            ? formattedDate(Number(deployment.createdAt))
            : timeSince(Number(deployment.createdAt) * 1000) + " ago"}
        </Typography>
      }
      sx={sx}
    >
      <Box
        sx={{
          flex: 1,
          display: "flex",
          alignItems: "flex-start",
        }}
      >
        {isRunning && (
          <CircularProgress
            size={12}
            color="error"
            variant="indeterminate"
            sx={iconProps}
          />
        )}
        {!isRunning && (
          <LensIcon color={isSuccess ? "success" : "error"} sx={iconProps} />
        )}
        <Box sx={{ flex: 1 }}>
          <CommitInfo deployment={deployment} showProject={showProject} />
        </Box>
      </Box>
      {showDeleteModal && (
        <ConfirmModal
          onConfirm={({ setLoading, setError }) => {
            setLoading(true);
            deleteForever({
              appId: deployment.appId,
              deploymentId: deployment.id,
            })
              .then(() => {
                setLoading(false);
                setShowDeleteModal(undefined);
                setRefreshToken(Date.now());
              })
              .catch(() => {
                setLoading(false);
                setError(
                  "Something went wrong while deleting deployment. Please try again.",
                );
              });
          }}
          onCancel={() => {
            setShowDeleteModal(undefined);
          }}
        >
          This will completely remove the deployment and the artifacts. This
          action is irreversible.
        </ConfirmModal>
      )}
      {showStopModal && (
        <ConfirmModal
          onConfirm={({ setLoading, setError }) => {
            setLoading(true);
            stopDeployment({
              envId: deployment.envId,
              deploymentId: deployment.id,
            })
              .then(() => {
                setLoading(false);
                setShowStopModal(undefined);
                setRefreshToken(Date.now());
              })
              .catch(() => {
                setLoading(false);
                setError(
                  "Something went wrong while stopping deployment. Please try again.",
                );
              });
          }}
          onCancel={() => {
            setShowStopModal(undefined);
          }}
        >
          This will stop the deployment.
        </ConfirmModal>
      )}
      {showPublishModal && (
        <PublishModal
          deployment={deployment}
          onUpdate={() => {
            setRefreshToken(Date.now());
          }}
          onClose={() => {
            setShowPublishModal(undefined);
          }}
        />
      )}
      {showManifest && (
        <ManifestModal
          deployment={deployment}
          onClose={() => {
            setShowManifest(undefined);
          }}
        />
      )}
    </CardRow>
  );
}
