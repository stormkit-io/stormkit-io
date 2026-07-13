import { SxProps } from "@mui/material/styles";
import Box from "@mui/material/Box";
import Typography from "@mui/material/Typography";
import Tooltip from "@mui/material/Tooltip";
import AnnouncementIcon from "@mui/icons-material/Announcement";

interface Props {
  placement?: "top" | "bottom" | "left" | "right";
  title?: string;
  subtitle?: string;
  sx?: SxProps;
}

export default function ActionRequired({
  title,
  subtitle,
  sx,
  placement = "top",
}: Props) {
  return (
    <Tooltip
      title={
        <Box sx={{ p: 1 }}>
          <Typography variant="subtitle1" sx={{ mb: subtitle ? 2 : 0 }}>
            {title || "Action required"}
          </Typography>
          {subtitle && <Typography variant="body2">{subtitle}</Typography>}
        </Box>
      }
      placement={placement}
      arrow
    >
      <AnnouncementIcon color="warning" sx={sx} />
    </Tooltip>
  );
}
