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
