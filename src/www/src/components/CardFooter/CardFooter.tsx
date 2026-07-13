import type { BoxProps } from "@mui/material";
import Box from "@mui/material/Box";

interface Props extends BoxProps {}

export default function CardFooter({ children, sx, ...rest }: Props) {
  return (
    <Box sx={{ textAlign: "right", color: "text.secondary", ...sx }} {...rest}>
      {children}
    </Box>
  );
}
