import type { TimeSpan } from "./index.d";
import Tooltip from "@mui/material/Tooltip";
import Typography from "@mui/material/Typography";

export function truncate(s: string) {
  if (s.length > 50) {
    return (
      <Tooltip title={s} arrow>
        <Typography>{s.substring(0, 50) + "..."}</Typography>
      </Tooltip>
    );
  }

  return s;
}

// channelLabels names the referrer channels of the analytics reports.
export const channelLabels: Record<string, string> = {
  search: "Search",
  ai: "AI assistants",
  social: "Social",
  email: "Email",
  referral: "Other sites",
  direct: "Direct",
  internal: "Within this site",
};

export const spanText = (ts: TimeSpan) =>
  ({
    "24h": "today (since midnight UTC)",
    "7d": "in the last 7 days",
    "30d": "in the last 30 days",
    "90d": "in the last 90 days",
    "365d": "in the last year",
  })[ts];
