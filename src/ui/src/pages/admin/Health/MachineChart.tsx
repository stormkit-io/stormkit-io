import Box from "@mui/material/Box";
import Typography from "@mui/material/Typography";
import {
  Area,
  AreaChart,
  CartesianGrid,
  ResponsiveContainer,
  Tooltip,
  YAxis,
} from "recharts";
import { useFetchHistory } from "./actions";

interface Props {
  target: string;
  refreshToken?: number;
}

interface Point {
  ts: number;
  cpu: number | null;
  memory: number;
}

/**
 * Renders the retained history for one machine. CPU is omitted for samples
 * where no rate could be derived (first sample, counter reset), so a restart
 * leaves a gap rather than a false zero.
 */
export default function MachineChart({ target, refreshToken }: Props) {
  const { samples, loading } = useFetchHistory({ target, refreshToken });

  if (loading || samples.length < 2) {
    return null;
  }

  const points: Point[] = samples
    .filter(s => s.reachable)
    .map(s => ({
      ts: s.ts,
      cpu: s.cpuValid ? Number(s.cpuPercent.toFixed(1)) : null,
      memory: s.memTotalBytes
        ? Number(
            (
              ((s.memTotalBytes - s.memAvailableBytes) / s.memTotalBytes) *
              100
            ).toFixed(1),
          )
        : 0,
    }));

  if (points.length < 2) {
    return null;
  }

  return (
    <Box sx={{ mt: 2 }}>
      <Typography sx={{ fontSize: 12, mb: 1, opacity: 0.6 }}>
        CPU and memory usage, last 24 hours (%)
      </Typography>
      <Box sx={{ height: 140 }}>
        <ResponsiveContainer width="100%" height="100%">
          <AreaChart data={points}>
            <CartesianGrid strokeDasharray="3 3" opacity={0.15} />
            <YAxis domain={[0, 100]} width={30} tick={{ fontSize: 10 }} />
            <Tooltip
              labelFormatter={(ts: number) =>
                new Date(ts * 1000).toLocaleString()
              }
            />
            <Area
              type="monotone"
              dataKey="cpu"
              name="CPU"
              stroke="#8884d8"
              fill="#8884d8"
              fillOpacity={0.2}
              connectNulls={false}
            />
            <Area
              type="monotone"
              dataKey="memory"
              name="Memory"
              stroke="#82ca9d"
              fill="#82ca9d"
              fillOpacity={0.2}
            />
          </AreaChart>
        </ResponsiveContainer>
      </Box>
    </Box>
  );
}
