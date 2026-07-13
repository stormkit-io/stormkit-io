import type { SxProps } from "@mui/material";
import Box from "@mui/material/Box";

interface Props {
  sx?: SxProps;
}

export default function Dot({ sx }: Props) {
  return (
    <Box component="span" sx={{ mx: 0.5, opacity: 0.5, ...sx }}>
      &#183;
    </Box>
  );
}
