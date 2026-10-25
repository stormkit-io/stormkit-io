import type { AnalyticsTotals, TimeSpan } from "./index.d";
import { useMemo, useState, useContext } from "react";
import {
  YAxis,
  AreaChart,
  Area,
  Tooltip,
  CartesianGrid,
  ResponsiveContainer,
} from "recharts";
import Box from "@mui/material/Box";
import ToggleButton from "@mui/material/ToggleButton";
import ToggleButtonGroup from "@mui/material/ToggleButtonGroup";
import Typography from "@mui/material/Typography";
import { green, grey, pink, red } from "@mui/material/colors";
import Card from "~/components/Card";
import CardHeader from "~/components/CardHeader";
import CardFooter from "~/components/CardFooter";
import { RootContext } from "~/pages/Root.context";
import { useFetchVisitors, useFetchVisitorsReport } from "./actions";

const seriesLabels: Record<string, string> = {
  total: "page views",
  unique: "visitors",
};

interface CustomTooltipProps {
  active?: boolean;
  payload?: any;
  label?: string;
}

// tooltipLabel names a chart point: the hour of an hourly point
// ("YYYY-MM-DD HH:MM", aggregated in UTC) or the date of a daily one.
export function tooltipLabel(name: string): string {
  const [date, hour] = name.split(" ");

  return hour ? `${hour} UTC` : date;
}

function CustomTooltip({ active, payload }: CustomTooltipProps) {
  if (active && payload && payload.length) {
    return (
      <Box
        sx={{
          p: 2,
          borderRadius: 1,
          bgcolor: "rgba(0,0,0,0.5)",
          minWidth: "200px",
          textAlign: "center",
        }}
      >
        <Typography
          sx={{
            borderBottom: `1px solid ${grey[900]}`,
            pb: 1,
            mb: 1,
          }}
        >
          {tooltipLabel(payload[0].payload.name)}
        </Typography>
        {payload.map((p: any) => (
          <Typography key={p.name}>
            {formatter.format(p.value)} {seriesLabels[p.name] || p.name}
          </Typography>
        ))}
      </Box>
    );
  }

  return null;
}

const timeSpans: { value: TimeSpan; label: string; text: string }[] = [
  { value: "24h", label: "24 hours", text: "24 hours" },
  { value: "7d", label: "7 Days", text: "7 days" },
  { value: "30d", label: "30 Days", text: "30 days" },
  { value: "90d", label: "90 Days", text: "90 days" },
  { value: "365d", label: "1 Year", text: "year" },
];

const formatter = new Intl.NumberFormat(undefined, {
  minimumFractionDigits: 0,
});

// formatDuration renders seconds as e.g. "45s" or "2m 05s".
export function formatDuration(seconds: number): string {
  if (seconds < 60) {
    return `${seconds}s`;
  }

  const minutes = Math.floor(seconds / 60);
  const rest = `${seconds % 60}`.padStart(2, "0");

  return `${minutes}m ${rest}s`;
}

// Change renders the change of a total against the previous period, or nothing
// when there is nothing to compare with.
export function Change({
  current,
  previous,
}: {
  current?: number;
  previous?: number;
}) {
  if (current === undefined || !previous) {
    return null;
  }

  const change = Math.round(((current - previous) / previous) * 100);

  return (
    <Box
      component="span"
      sx={{ ml: 0.5, fontSize: 12, color: change >= 0 ? green[400] : red[300] }}
    >
      {change >= 0 ? "+" : ""}
      {change}%
    </Box>
  );
}

interface SummaryProps {
  totals: AnalyticsTotals;
  previous?: AnalyticsTotals;
  span: string;
}

function Summary({ totals, previous, span }: SummaryProps) {
  return (
    <>
      <Box component="span" sx={{ color: pink[300] }}>
        {formatter.format(totals.pageViews || 0)}
      </Box>{" "}
      page views
      <Change current={totals.pageViews} previous={previous?.pageViews} />
      {totals.visitors !== undefined && (
        <>
          {" · "}
          <Box component="span" sx={{ color: pink[300] }}>
            {formatter.format(totals.visitors)}
          </Box>{" "}
          visitors
          <Change current={totals.visitors} previous={previous?.visitors} />
        </>
      )}
      {totals.avgTimeOnPage !== undefined && (
        <>
          {" · "}
          <Box component="span" sx={{ color: pink[300] }}>
            {formatDuration(totals.avgTimeOnPage)}
          </Box>{" "}
          average time on page
          <Change
            current={totals.avgTimeOnPage}
            previous={previous?.avgTimeOnPage}
          />
        </>
      )}{" "}
      in the last {span}
    </>
  );
}

interface Props {
  environment: Environment;
  ts: TimeSpan;
  domain: Domain;
  onTimeSpanChange: (t: TimeSpan) => void;
}

export default function Visitors({
  environment,
  onTimeSpanChange,
  ts,
  domain,
}: Props) {
  const { mode } = useContext(RootContext);
  const [display, setDisplay] = useState<"unique" | "total" | "all">("all");
  const hourly = ts === "24h";

  // The last 24 hours are charted per hour from the hourly aggregate; longer
  // spans read the daily report, which also has the previous period.
  const hours = useFetchVisitors({
    envId: environment.id!,
    domainId: hourly ? domain?.id : undefined,
  });

  const days = useFetchVisitorsReport({
    envId: environment.id!,
    domainId: hourly ? undefined : domain?.id,
    ts: hourly ? "7d" : ts,
  });

  const visitors = useMemo(() => {
    if (hourly) {
      return hours.visitors;
    }

    return (days.report?.series || []).map(day => ({
      name: day.date,
      total: day.pageViews,
      unique: day.visitors || 0,
    }));
  }, [hourly, hours.visitors, days.report]);

  const totals: AnalyticsTotals = useMemo(() => {
    if (!hourly && days.report) {
      return days.report.totals;
    }

    // Hourly unique visitors cannot be summed into a day's unique visitors,
    // so the last 24 hours only total their page views.
    return {
      pageViews: visitors.reduce((sum, v) => sum + v.total, 0),
    } as AnalyticsTotals;
  }, [hourly, days.report, visitors]);

  const isDark = mode === "dark";
  const largest = Math.max(totals.pageViews || 0, totals.visitors || 0);
  const span = timeSpans.find(t => t.value === ts)?.text || ts;

  return (
    <Card
      error={hourly ? hours.error : days.error}
      loading={hourly ? hours.loading : days.loading}
    >
      <CardHeader
        title="Visitors"
        subtitle={
          <Summary
            totals={totals}
            previous={hourly ? undefined : days.report?.previous?.totals}
            span={span}
          />
        }
        actions={
          <ToggleButtonGroup
            value={ts}
            exclusive
            sx={{ bgcolor: "container.paper" }}
            onChange={(_, val) => {
              if (val !== null) {
                onTimeSpanChange(val);
              }
            }}
            aria-label="Time span"
          >
            {timeSpans.map(t => (
              <ToggleButton
                key={t.value}
                value={t.value}
                aria-label={t.text}
                size="small"
                sx={{
                  fontSize: 12,
                  textTransform: "capitalize",
                  color: "text.primary",
                }}
              >
                {t.label}
              </ToggleButton>
            ))}
          </ToggleButtonGroup>
        }
      />
      <Box sx={{ height: 300, mb: 4 }}>
        <ResponsiveContainer width="100%" height="100%">
          <AreaChart
            data-testid="area-chart"
            data={visitors}
            margin={{
              top: 10,
              right: 5,
              left: largest > 1000 ? -15 : largest > 100 ? -25 : -28,
              bottom: 10,
            }}
          >
            <CartesianGrid
              horizontalPoints={[0]}
              stroke={isDark ? "#181329" : grey[400]}
            />
            <Tooltip content={<CustomTooltip />} />
            <YAxis
              tick={{
                fill: isDark ? grey[500] : grey[800],
                fontFamily: "Roboto, Helvetica, Arial, sans-serif",
                fontWeight: 400,
                fontSize: 12,
              }}
            />
            <Area
              hide={display === "unique"}
              type="linear"
              dataKey="total"
              stroke="#6048b0a9"
              fill="#1c499ca9"
              dot
            />
            <Area
              hide={display === "total"}
              type="linear"
              dataKey="unique"
              stroke="#6048b0a9"
              fill="#a048b0a9"
              dot
            />
          </AreaChart>
        </ResponsiveContainer>
        <Box sx={{ justifyContent: "center", display: "flex", width: "100%" }}>
          <Typography
            sx={{
              color: "#1c499c",
              mr: 2,
              cursor: "pointer",
            }}
            onClick={() => {
              setDisplay(display === "total" ? "all" : "total");
            }}
          >
            Page views
          </Typography>
          <Typography
            sx={{ color: "#7a3782", cursor: "pointer" }}
            onClick={() => {
              setDisplay(display === "unique" ? "all" : "unique");
            }}
          >
            Visitors
          </Typography>
        </Box>
      </Box>
      <CardFooter sx={{ textAlign: "left" }}>
        <Typography sx={{ fontSize: 12 }}>
          Bots are excluded from these statistics.{" "}
          {hourly
            ? "Visitors are counted per hour, so a person active in two hours counts twice."
            : "Visitors are counted per day, so a person who visits on two days counts twice. Days end at midnight UTC; today is in the 24 hours view."}
        </Typography>
      </CardFooter>
    </Card>
  );
}
