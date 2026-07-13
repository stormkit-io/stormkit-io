import Box from "@mui/material/Box";
import Typography from "@mui/material/Typography";
import Alert from "@mui/material/Alert";
import Chip from "@mui/material/Chip";
import Tooltip from "@mui/material/Tooltip";
import Skeleton from "@mui/material/Skeleton";
import TrendingUpIcon from "@mui/icons-material/TrendingUp";
import TrendingDownIcon from "@mui/icons-material/TrendingDown";
import TrendingFlatIcon from "@mui/icons-material/TrendingFlat";
import { formatNumber } from "~/utils/helpers/string";
import { calculateChange } from "~/utils/helpers/stats";
import { useFetchTeamStats } from "../actions";
import UpgradeButton from "~/components/UpgradeButton";

interface Props {
  teamId?: string;
}

const formatDuration = (seconds: number): string => {
  if (seconds < 60) {
    return `${seconds.toFixed(2)}s`;
  }

  const minutes = Math.floor(seconds / 60);
  const remainingSeconds = Math.floor(seconds % 60);

  return remainingSeconds > 0
    ? `${minutes}m ${remainingSeconds}s`
    : `${minutes}m`;
};

interface StatBoxProps {
  header: React.ReactNode;
  tooltip: string;
  change?: number;
  changeType: "percentage" | "absolute";
  changeDirection?: "inverse" | "normal";
  desc: string;
  loading?: boolean;
}

type Trend = "unknown" | "up" | "down" | "same";

const StatBox = ({
  desc,
  header,
  change,
  changeType,
  changeDirection = "normal",
  tooltip,
  loading,
}: StatBoxProps) => {
  let operator = "%";

  if (changeType === "absolute" && typeof change !== "undefined") {
    operator = change > 0 ? "+" : "-";
  }

  const trend: Trend =
    typeof change === "undefined" || loading
      ? "unknown"
      : change > 0
      ? "up"
      : change < 0
      ? "down"
      : "same";

  const icons = {
    unknown: undefined,
    same: <TrendingFlatIcon />,
    up:
      changeDirection === "normal" ? <TrendingUpIcon /> : <TrendingDownIcon />,
    down:
      changeDirection === "normal" ? <TrendingDownIcon /> : <TrendingUpIcon />,
  };

  const bgcolor = {
    unknown: "rgba(128, 128, 128, 0.3)",
    up: "rgba(48, 127, 51, 0.45)",
    down: "rgba(244, 67, 54, 0.1)",
    same: "rgba(128, 128, 128, 0.3)",
  };

  const borderColor = {
    unknown: "rgba(128, 128, 128, 0.7)",
    up: "rgba(48, 127, 51, 0.4)",
    down: "#F44336",
    same: "rgba(128, 128, 128, 0.7)",
  };

  const label = {
    unknown: "No data to compare",
    up: `${operator}${Math.abs(change || 0)} this month`,
    down: `${operator}${Math.abs(change || 0)} this month`,
    same: "No change",
  };

  return (
    <Box
      sx={{
        bgcolor: "container.transparent",
        border: "1px solid rgba(255,255,255,0.1)",
        p: 4,
        textAlign: "center",
        borderRadius: 1,
        position: "relative",
      }}
    >
      <Box>
        <Typography
          sx={{
            fontSize: 32,
            mr: 1,
            fontWeight: 600,
            lineHeight: 1.25,
            mb: 1,
          }}
        >
          {loading ? <Skeleton data-testid="skeleton" /> : header}
        </Typography>
        <Typography color="text.secondary" sx={{ lineHeight: 1.25 }}>
          {desc}
        </Typography>
      </Box>
      <Tooltip arrow title={<Box sx={{ p: 2 }}>{tooltip}</Box>}>
        <Typography sx={{ lineHeight: 1 }}>
          <Chip
            size="small"
            variant="filled"
            icon={icons[trend]}
            component="span"
            label={loading ? undefined : label[trend]}
            sx={{
              border: "1px solid",
              backgroundColor: bgcolor[trend],
              borderColor: borderColor[trend],
              color: "white",
              display: "flex",
              mt: 2,
            }}
          />
        </Typography>
      </Tooltip>
    </Box>
  );
};

export default function TeamStats({ teamId }: Props) {
  const { stats, loading, error, paymentRequired } = useFetchTeamStats({
    teamId,
  });

  if (paymentRequired) {
    return (
      <Alert severity="info" sx={{ mx: 4, mb: 2 }}>
        <Typography>
          Team stats are available on the Enterprise plan.{" "}
          <UpgradeButton fullWidth={false} variant="text" /> to get access.
        </Typography>
      </Alert>
    );
  }

  const totalNewApps =
    (stats?.totalApps.new || 0) - (stats?.totalApps.deleted || 0);

  const deploymentChange = calculateChange(
    stats?.totalDeployments.current,
    stats?.totalDeployments.previous
  );

  const avgDurationChange = calculateChange(
    stats?.avgDeploymentDuration.previous,
    stats?.avgDeploymentDuration.current
  );

  const requestChange = calculateChange(
    stats?.totalRequests.current,
    stats?.totalRequests.previous
  );

  const avgDuration = formatDuration(stats?.avgDeploymentDuration.current || 0);

  const totalRequests = formatNumber(stats?.totalRequests.current || 0);

  return (
    <>
      {error && (
        <Alert severity="error" sx={{ mx: 4, mb: 2 }}>
          {error}
        </Alert>
      )}
      <Box
        sx={{
          display: "grid",
          gridTemplateColumns: { xs: "repeat(1, 1fr)", md: "repeat(4, 1fr)" },
          gap: 2,
          p: 4,
          pt: 0,
          position: "relative",
        }}
      >
        <StatBox
          desc="Apps"
          tooltip="Number of total apps in this team."
          header={stats?.totalApps.total || 0}
          change={totalNewApps}
          changeType="absolute"
          loading={loading}
        />
        <StatBox
          desc="Deployments"
          tooltip="Total number of deployments in the last 30 days. Data is compared against previous 30 days."
          header={stats?.totalDeployments.current || 0}
          change={deploymentChange}
          changeType="percentage"
          loading={loading}
        />
        <StatBox
          desc="Avg. deployment duration"
          tooltip="Average time taken for deployments in the last 30 days. Data is compared against previous 30 days."
          header={avgDuration}
          change={avgDurationChange}
          changeType="percentage"
          changeDirection="inverse"
          loading={loading}
        />
        <StatBox
          tooltip="Total requests accross all domains in this team in the last 30 days. Data is compared against previous 30 days."
          desc="Requests"
          header={totalRequests}
          change={requestChange}
          changeType="percentage"
          loading={loading}
        />
      </Box>
    </>
  );
}
