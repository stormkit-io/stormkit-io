import type { SxProps } from "@mui/material";
import Typography from "@mui/material/Typography";
import Link from "@mui/material/Link";

export interface Path {
  path: string;
  icon?: React.ReactNode;
  text: React.ReactNode;
  isActive?: boolean;
}

interface Props {
  item: Path;
  sx?: SxProps;
}

export default function MenuLink({ item, sx }: Props) {
  return (
    <Typography sx={{ display: "inline-block" }}>
      <Link
        key={item.path}
        href={item.path}
        sx={{
          cursor: "pointer",
          px: { xs: 1, md: 2 },
          py: 0.5,
          display: "inline-flex",
          position: "relative",
          alignItems: "center",
          borderRadius: 1,
          transition: "background-color 0.2s ease, color 0.2s ease",
          bgcolor: item.isActive ? "rgba(81, 81, 81, 0.24)" : undefined,
          ":hover": {
            opacity: 1,
            bgcolor: "rgba(81, 81, 81, 0.50)",
            color: "text.primary",
          },
          ...sx,
        }}
      >
        {item.icon}
        {item.text}
      </Link>
    </Typography>
  );
}
