import { useEffect, useState } from "react";
import api from "~/utils/api/Api";

export interface Filesystem {
  mountpoint: string;
  device: string;
  fsType?: string;
  sizeBytes: number;
  availBytes: number;
}

export interface Sample {
  ts: number;
  target: string;
  reachable: boolean;
  error?: string;
  cpuPercent: number;
  cpuValid: boolean;
  cpuCores: number;
  memTotalBytes: number;
  memAvailableBytes: number;
  load1: number;
  load5: number;
  load15: number;
  filesystems?: Filesystem[];
  netReceiveBytes: number;
  netTransmitBytes: number;
  bootTime: number;
}

export interface Machine {
  host: string;
  services: string[];
  manual: boolean;
  /** Null until the scraper has run, which is normal in the first minute. */
  sample: Sample | null;
}

export interface TableSize {
  name: string;
  bytes: number;
}

export interface Dependencies {
  ts: number;
  postgres: {
    reachable: boolean;
    error?: string;
    latencyMs: number;
    databaseBytes: number;
    connections: number;
    maxConnections: number;
    largestTables?: TableSize[];
  };
  redis: {
    reachable: boolean;
    error?: string;
    latencyMs: number;
    usedMemoryBytes: number;
  };
}

export interface PoolStats {
  open: number;
  inUse: number;
  idle: number;
  waitCount: number;
  waitDurationMs: number;
  maxOpenConnections: number;
}

export interface Metrics {
  machines: Machine[];
  dependencies: Dependencies | null;
  pool: PoolStats;
  retentionHours: number;
}

interface UseFetchMetricsProps {
  refreshToken?: number;
}

export const useFetchMetrics = ({
  refreshToken,
}: UseFetchMetricsProps = {}) => {
  const [metrics, setMetrics] = useState<Metrics>();
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string>();

  useEffect(() => {
    api
      .fetch<Metrics>("/admin/metrics")
      .then(m => {
        setMetrics(m);
        setError(undefined);
      })
      .catch(() => {
        setError(
          "Something went wrong while fetching system health. Please try again later.",
        );
      })
      .finally(() => {
        setLoading(false);
      });
  }, [refreshToken]);

  return { metrics, loading, error };
};

interface UseFetchHistoryProps {
  target?: string;
  refreshToken?: number;
}

export const useFetchHistory = ({
  target,
  refreshToken,
}: UseFetchHistoryProps) => {
  const [samples, setSamples] = useState<Sample[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    if (!target) {
      setLoading(false);
      return;
    }

    setLoading(true);

    api
      .fetch<{ samples: Sample[] }>(
        `/admin/metrics/history?target=${encodeURIComponent(target)}`,
      )
      .then(res => {
        setSamples(res.samples || []);
      })
      .catch(() => {
        setSamples([]);
      })
      .finally(() => {
        setLoading(false);
      });
  }, [target, refreshToken]);

  return { samples, loading };
};

export const useFetchTargets = ({
  refreshToken,
}: UseFetchMetricsProps = {}) => {
  const [targets, setTargets] = useState<string[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    api
      .fetch<{ targets: string[] }>("/admin/metrics/targets")
      .then(res => {
        setTargets(res.targets || []);
      })
      .catch(() => {
        setTargets([]);
      })
      .finally(() => {
        setLoading(false);
      });
  }, [refreshToken]);

  return { targets, loading };
};

export const updateTargets = (targets: string[]) => {
  return api.put<{ targets: string[] }>("/admin/metrics/targets", { targets });
};

const UNITS = ["B", "KB", "MB", "GB", "TB", "PB"];

/** Formats bytes for display, e.g. 1536 => "1.5 KB". */
export const formatBytes = (bytes: number): string => {
  if (!bytes || bytes < 0) {
    return "0 B";
  }

  let value = bytes;
  let unit = 0;

  while (value >= 1024 && unit < UNITS.length - 1) {
    value = value / 1024;
    unit = unit + 1;
  }

  return `${value >= 10 || unit === 0 ? Math.round(value) : value.toFixed(1)} ${
    UNITS[unit]
  }`;
};

/** Formats an uptime in seconds as a coarse human duration. */
export const formatUptime = (bootTime: number): string => {
  if (!bootTime) {
    return "-";
  }

  const seconds = Math.floor(Date.now() / 1000) - bootTime;

  if (seconds < 60) {
    return "just booted";
  }

  const days = Math.floor(seconds / 86400);
  const hours = Math.floor((seconds % 86400) / 3600);
  const minutes = Math.floor((seconds % 3600) / 60);

  if (days > 0) {
    return `${days}d ${hours}h`;
  }

  if (hours > 0) {
    return `${hours}h ${minutes}m`;
  }

  return `${minutes}m`;
};

/** The command that installs the exporter this page reads from. */
export const EXPORTER_COMMAND = [
  "docker run -d --name node-exporter --pid host --net host \\",
  "  --restart unless-stopped -v /:/host:ro,rslave \\",
  "  prom/node-exporter --path.rootfs=/host",
].join("\n");
