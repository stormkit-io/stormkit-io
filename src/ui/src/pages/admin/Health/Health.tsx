import { useEffect, useState } from "react";
import Box from "@mui/material/Box";
import Chip from "@mui/material/Chip";
import LinearProgress from "@mui/material/LinearProgress";
import Tooltip from "@mui/material/Tooltip";
import Typography from "@mui/material/Typography";
import Card from "~/components/Card";
import CardHeader from "~/components/CardHeader";
import CardRow from "~/components/CardRow";
import CopyBox from "~/components/CopyBox";
import MachineChart from "./MachineChart";
import ManualTargets from "./ManualTargets";
import type { Dependencies, Filesystem, Machine, PoolStats } from "./actions";
import {
  EXPORTER_COMMAND,
  formatBytes,
  formatUptime,
  useFetchMetrics,
} from "./actions";

/** Matches the scrape interval; polling faster only re-reads the same sample. */
const POLL_INTERVAL = 60 * 1000;

const usageColor = (percent: number) => {
  if (percent >= 90) {
    return "error";
  }

  if (percent >= 75) {
    return "warning";
  }

  return "primary";
};

interface UsageProps {
  label: string;
  used: number;
  total: number;
  caption?: string;
}

function Usage({ label, used, total, caption }: UsageProps) {
  const percent = total > 0 ? Math.min(100, (used / total) * 100) : 0;

  return (
    <Box sx={{ mb: 2 }}>
      <Box sx={{ display: "flex", justifyContent: "space-between", mb: 0.5 }}>
        <Typography sx={{ fontSize: 12 }}>{label}</Typography>
        <Typography sx={{ fontSize: 12, opacity: 0.6 }}>
          {caption || `${formatBytes(used)} / ${formatBytes(total)}`}
        </Typography>
      </Box>
      <LinearProgress
        variant="determinate"
        value={percent}
        color={usageColor(percent)}
        sx={{ height: 6, borderRadius: 1 }}
      />
    </Box>
  );
}

function FilesystemRow({ fs }: { fs: Filesystem }) {
  const used = Math.max(0, fs.sizeBytes - fs.availBytes);

  return (
    <Usage
      label={`${fs.mountpoint} (${fs.device})`}
      used={used}
      total={fs.sizeBytes}
      caption={`${formatBytes(used)} / ${formatBytes(fs.sizeBytes)} used`}
    />
  );
}

function MachineCard({
  machine,
  refreshToken,
}: {
  machine: Machine;
  refreshToken?: number;
}) {
  const { sample } = machine;

  return (
    <Card sx={{ mb: 2 }}>
      <CardHeader
        title={machine.host}
        subtitle={
          machine.services.length
            ? machine.services.join(", ")
            : "No Stormkit services on this machine"
        }
      />
      <CardRow>
        {!sample && (
          <Typography sx={{ fontSize: 14, opacity: 0.6 }}>
            Waiting for the first sample. This can take up to a minute.
          </Typography>
        )}
        {sample && !sample.reachable && (
          <Box>
            <Chip label="Unreachable" color="error" size="small" />
            <Typography sx={{ fontSize: 14, mt: 1, opacity: 0.8 }}>
              node_exporter is not answering on this machine.
            </Typography>
            {sample.error && (
              <Typography sx={{ fontSize: 12, mt: 0.5, opacity: 0.6 }}>
                {sample.error}
              </Typography>
            )}
          </Box>
        )}
        {sample?.reachable && (
          <Box>
            <Box sx={{ display: "flex", gap: 2, mb: 2, flexWrap: "wrap" }}>
              <Chip
                size="small"
                label={`${sample.cpuCores} core${
                  sample.cpuCores === 1 ? "" : "s"
                }`}
              />
              <Chip
                size="small"
                label={`up ${formatUptime(sample.bootTime)}`}
              />
              <Tooltip title="1, 5 and 15 minute load average">
                <Chip
                  size="small"
                  label={`load ${sample.load1.toFixed(2)} / ${sample.load5.toFixed(
                    2,
                  )} / ${sample.load15.toFixed(2)}`}
                />
              </Tooltip>
            </Box>

            <Usage
              label="CPU"
              used={sample.cpuValid ? sample.cpuPercent : 0}
              total={100}
              caption={
                sample.cpuValid
                  ? `${sample.cpuPercent.toFixed(1)}%`
                  : "awaiting a second sample"
              }
            />

            <Usage
              label="Memory"
              used={sample.memTotalBytes - sample.memAvailableBytes}
              total={sample.memTotalBytes}
            />

            {sample.filesystems?.map(fs => (
              <FilesystemRow key={fs.mountpoint} fs={fs} />
            ))}

            <MachineChart target={machine.host} refreshToken={refreshToken} />

            {machine.processes?.length ? (
              <Box sx={{ mt: 2 }}>
                <Typography sx={{ fontSize: 12, mb: 0.5, opacity: 0.6 }}>
                  Stormkit processes on this machine
                </Typography>
                {machine.processes.map(p => (
                  <Box
                    key={p.instanceId}
                    sx={{ display: "flex", justifyContent: "space-between" }}
                  >
                    <Typography sx={{ fontSize: 12, opacity: 0.8 }}>
                      {p.service}
                    </Typography>
                    <Typography sx={{ fontSize: 12, opacity: 0.6 }}>
                      {formatBytes(p.rssBytes || p.heapBytes)} &middot;{" "}
                      {p.goroutines} goroutines
                    </Typography>
                  </Box>
                ))}
              </Box>
            ) : null}
          </Box>
        )}
      </CardRow>
    </Card>
  );
}

function DependencyCard({
  dependencies,
  pool,
}: {
  dependencies: Dependencies | null;
  pool?: PoolStats;
}) {
  return (
    <Card sx={{ mb: 2 }}>
      <CardHeader title="Dependencies" subtitle="PostgreSQL and Redis" />
      <CardRow>
        {!dependencies && (
          <Typography sx={{ fontSize: 14, opacity: 0.6 }}>
            Waiting for the first sample.
          </Typography>
        )}
        {dependencies && (
          <Box>
            <Box sx={{ mb: 3 }}>
              <Box
                sx={{ display: "flex", gap: 2, mb: 1, alignItems: "center" }}
              >
                <Typography sx={{ fontSize: 14 }}>PostgreSQL</Typography>
                <Chip
                  size="small"
                  color={dependencies.postgres.reachable ? "success" : "error"}
                  label={dependencies.postgres.reachable ? "Up" : "Down"}
                />
                {dependencies.postgres.reachable && (
                  <Typography sx={{ fontSize: 12, opacity: 0.6 }}>
                    {dependencies.postgres.latencyMs.toFixed(1)} ms &middot;{" "}
                    {formatBytes(dependencies.postgres.databaseBytes)} &middot;{" "}
                    {dependencies.postgres.connections} /{" "}
                    {dependencies.postgres.maxConnections} connections
                  </Typography>
                )}
              </Box>

              {dependencies.postgres.error && (
                <Typography sx={{ fontSize: 12, opacity: 0.6 }}>
                  {dependencies.postgres.error}
                </Typography>
              )}

              {dependencies.postgres.largestTables?.map(table => (
                <Box
                  key={table.name}
                  sx={{ display: "flex", justifyContent: "space-between" }}
                >
                  <Typography sx={{ fontSize: 12, opacity: 0.8 }}>
                    {table.name}
                  </Typography>
                  <Typography sx={{ fontSize: 12, opacity: 0.6 }}>
                    {formatBytes(table.bytes)}
                  </Typography>
                </Box>
              ))}
            </Box>

            <Box sx={{ display: "flex", gap: 2, alignItems: "center" }}>
              <Typography sx={{ fontSize: 14 }}>Redis</Typography>
              <Chip
                size="small"
                color={dependencies.redis.reachable ? "success" : "error"}
                label={dependencies.redis.reachable ? "Up" : "Down"}
              />
              {dependencies.redis.reachable && (
                <Typography sx={{ fontSize: 12, opacity: 0.6 }}>
                  {dependencies.redis.latencyMs.toFixed(1)} ms &middot;{" "}
                  {formatBytes(dependencies.redis.usedMemoryBytes)} used
                </Typography>
              )}
            </Box>

            {pool && (
              <Typography sx={{ fontSize: 12, mt: 2, opacity: 0.6 }}>
                This instance's pool: {pool.inUse} in use, {pool.idle} idle,{" "}
                {pool.waitCount} waits
              </Typography>
            )}
          </Box>
        )}
      </CardRow>
    </Card>
  );
}

/**
 * Shown when nothing is reporting. It is an instruction rather than an error:
 * on a fresh instance no exporter exists yet, which is expected.
 */
function SetupGuide() {
  return (
    <Card sx={{ mb: 2 }}>
      <CardHeader
        title="No machines are reporting yet"
        subtitle="Stormkit reads machine stats from node_exporter. Run this on each machine you want to monitor."
      />
      <CardRow>
        <CopyBox value={EXPORTER_COMMAND} multiline maxRows={4} fullWidth />
        <Typography sx={{ fontSize: 12, mt: 2, opacity: 0.6 }}>
          Machines running Stormkit are picked up automatically. Machines
          without a Stormkit process can be added under manual targets.
        </Typography>
      </CardRow>
    </Card>
  );
}

export default function Health() {
  const [refreshToken, setRefreshToken] = useState(0);
  const { metrics, loading, error } = useFetchMetrics({ refreshToken });

  useEffect(() => {
    const tick = () => {
      // Polling a hidden tab wastes a scrape-sized round trip on nobody.
      if (document.visibilityState === "visible") {
        setRefreshToken(Date.now());
      }
    };

    const interval = setInterval(tick, POLL_INTERVAL);

    return () => clearInterval(interval);
  }, []);

  const machines = metrics?.machines || [];
  const hasReporting = machines.some(m => m.sample?.reachable);

  return (
    <Box>
      <Card loading={loading && !metrics} error={error} sx={{ mb: 2 }}>
        <CardHeader
          title="Health"
          subtitle={`Machine resources over the last ${
            metrics?.retentionHours || 24
          } hours`}
        />
      </Card>

      {!loading && !hasReporting && <SetupGuide />}

      {machines.map(machine => (
        <MachineCard
          key={machine.host}
          machine={machine}
          refreshToken={refreshToken}
        />
      ))}

      <DependencyCard
        dependencies={metrics?.dependencies || null}
        pool={metrics?.pool}
      />

      <ManualTargets onUpdate={() => setRefreshToken(Date.now())} />
    </Box>
  );
}
