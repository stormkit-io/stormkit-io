import Box from "@mui/material/Box";
import Typography from "@mui/material/Typography";
import IconButton from "@mui/material/IconButton";
import CloseIcon from "@mui/icons-material/Close";
import { alpha } from "@mui/material/styles";

interface Props {
  label: string;
  value?: string;
  testId: string;
  /** A filter whose key is chosen but whose value is still being entered. */
  pending?: boolean;
  onDelete: () => void;
}

export default function Token({
  label,
  value,
  testId,
  pending,
  onDelete,
}: Props) {
  return (
    <Box
      data-testid={testId}
      sx={theme => ({
        display: "flex",
        alignItems: "center",
        gap: 0.5,
        pl: 1,
        pr: 0.25,
        py: 0.25,
        borderRadius: 1,
        maxWidth: "100%",
        bgcolor: alpha(
          theme.palette.primary.main,
          pending ? 0.28 : theme.palette.mode === "dark" ? 0.18 : 0.1,
        ),
        border: "1px solid",
        borderColor: alpha(theme.palette.primary.main, pending ? 0.9 : 0.45),
      })}
    >
      <Typography
        variant="caption"
        sx={{ opacity: 0.7, whiteSpace: "nowrap", lineHeight: 1.6 }}
      >
        {label}:
      </Typography>
      {value && (
        <Typography
          variant="caption"
          sx={{
            fontWeight: 600,
            lineHeight: 1.6,
            overflow: "hidden",
            textOverflow: "ellipsis",
            whiteSpace: "nowrap",
          }}
        >
          {value}
        </Typography>
      )}
      <IconButton
        size="small"
        aria-label={`Remove ${label} filter`}
        sx={{ p: 0.25 }}
        onClick={e => {
          e.stopPropagation();
          onDelete();
        }}
      >
        <CloseIcon sx={{ fontSize: 13 }} />
      </IconButton>
    </Box>
  );
}
