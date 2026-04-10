import { useEffect, useState, useMemo } from "react";
import RefreshIcon from "@mui/icons-material/RefreshOutlined";
import DeleteIcon from "@mui/icons-material/DeleteOutlined";
import StopIcon from "@mui/icons-material/StopCircle";
import PreviewIcon from "@mui/icons-material/Preview";
import ArticleIcon from "@mui/icons-material/Article";
import ManifestIcon from "@mui/icons-material/ReceiptLong";
import api from "~/utils/api/Api";

interface DeleteForeverProps {
  appId: string;
  deploymentId: string;
}

export const deleteForever = ({
  appId,
  deploymentId,
}: DeleteForeverProps): Promise<void> => {
  return api.delete(`/app/deploy`, { deploymentId, appId });
};

interface PublishDeploymentsProps {
  appId: string;
  envId: string;
  percentages: Record<string, number>;
}

interface PublishInfo {
  deploymentId: string;
  percentage: number;
}

export const publishDeployments = ({
  appId,
  percentages,
  envId,
}: PublishDeploymentsProps): Promise<void> => {
  const publish: Array<PublishInfo> = [];
  let total = 0;

  Object.keys(percentages).forEach(deploymentId => {
    const percentage = percentages[deploymentId];
    total = total + percentage;

    publish.push({
      percentage,
      deploymentId,
    });
  });

  if (total !== 100) {
    return Promise.reject(
      `The sum of percentages has be to 100. Currently it is ${total}.`,
    );
  }

  return api.post(`/app/deployments/publish`, {
    appId,
    envId,
    publish,
  });
};

interface FetchManifestProps {
  deploymentId: string;
  appId: string;
}

interface FetchManifestReturnValue {
  manifest: Manifest;
  loading: boolean;
  error: string | null;
}

export const useFetchManifest = ({
  appId,
  deploymentId,
}: FetchManifestProps): FetchManifestReturnValue => {
  const [manifest, setManifest] = useState<Manifest>({});
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    setLoading(true);
    setError(null);

    api
      .fetch<{ manifest: Manifest }>(`/app/${appId}/manifest/${deploymentId}`, {
        method: "GET",
      })
      .then(data => setManifest(data.manifest))
      .catch(e => setError(e))
      .finally(() => setLoading(false));
  }, []);

  return {
    loading,
    error,
    manifest,
  };
};

interface StopDeploymentProps {
  envId: string;
  deploymentId: string;
}

export const stopDeployment = ({
  envId,
  deploymentId,
}: StopDeploymentProps): Promise<void> => {
  return api.post(`/v1/deployments/${deploymentId}/stop`, { envId });
};

interface RestartDeploymentProps {
  envId: string;
  deploymentId: string;
}

export const restartDeployment = ({
  envId,
  deploymentId,
}: RestartDeploymentProps): Promise<void> => {
  return api.post(`/v1/deployments/${deploymentId}/restart`, { envId });
};

interface WithMenuItemsProps {
  omittedItems: Array<"view-details">;
  deployment: DeploymentV2;
  onPublishClick: () => void;
  onManifestClick: () => void;
  onStateChangeClick: () => void;
  onRestartClick: () => void;
}

export const useWithMenuItems = ({
  omittedItems = [],
  deployment,
  onPublishClick,
  onManifestClick,
  onStateChangeClick,
  onRestartClick,
}: WithMenuItemsProps) => {
  return useMemo(() => {
    const items = [];

    items.push({
      text: "Publish",
      className: "text-green-50",
      disabled: deployment.status !== "success",
      onClick: onPublishClick,
    });

    if (!omittedItems.includes("view-details")) {
      items.push({
        text: "View details",
        href: deployment.detailsUrl,
      });
    }

    items.push(
      {
        text: "Manifest",
        icon: <ManifestIcon />,
        disabled: deployment.status !== "success",
        onClick: onManifestClick,
      },
      {
        text: "Runtime logs",
        icon: <ArticleIcon />,
        disabled: deployment.status !== "success",
        href: `${deployment.detailsUrl}/runtime-logs`,
      },
      {
        text: "Preview",
        icon: <PreviewIcon />,
        disabled: deployment.status !== "success",
        href: deployment.previewUrl,
      },
      {
        text: "Restart",
        icon: <RefreshIcon />,
        onClick: onRestartClick,
        disabled: deployment.status !== "failed",
      },
      {
        text: deployment.status === "running" ? "Stop" : "Delete",
        icon: deployment.status === "running" ? <StopIcon /> : <DeleteIcon />,
        onClick: onStateChangeClick,
      },
    );

    return items;
  }, [omittedItems]);
};
