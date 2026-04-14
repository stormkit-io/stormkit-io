import type { SxProps } from "@mui/material";
import Typography from "@mui/material/Typography";
import Link from "@mui/material/Link";
import ChevronRightIcon from "@mui/icons-material/ChevronRight";
import Dot from "~/components/Dot";

export interface Path {
  path: string;
  icon?: React.ReactNode;
  text: React.ReactNode;
  isActive?: boolean;
  children?: Omit<Path, "children">[];
}

interface Props {
  item: Path;
  inline?: boolean;
  dot?: boolean;
  sx?: SxProps;
}

export default function MenuLink({ item, sx, inline, dot = false }: Props) {
  return (
    <Typography
      component="div"
      sx={{ display: inline ? "inline-block" : "block" }}
    >
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
          justifyContent: "space-between",
          borderRadius: 1,
          transition: "background-color 0.2s ease, color 0.2s ease",
          bgcolor:
            item.isActive && !item.children?.length
              ? "rgba(81, 81, 81, 0.24)"
              : undefined,
          ":hover": {
            opacity: 1,
            bgcolor: "rgba(81, 81, 81, 0.50)",
            color: "text.primary",
          },
          ...sx,
        }}
      >
        <span>
          {item.icon ?? (dot ? <Dot sx={{ mr: 2 }} /> : null)}
          {item.text}
        </span>
        {item.children && (
          <ChevronRightIcon
            sx={{
              ml: 2,
              color: "text.secondary",
              fontSize: 18,
              rotate: item.isActive ? "90deg" : "0deg",
            }}
          />
        )}
      </Link>
    </Typography>
  );
}
