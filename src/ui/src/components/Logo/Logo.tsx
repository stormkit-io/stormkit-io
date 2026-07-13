import type { SxProps } from "@mui/material";
import Box from "@mui/material/Box";
import logoText from "~/assets/logos/stormkit-logo-text-h-white.svg";
import logoIcon from "~/assets/logos/stormkit-logo-circle.svg";

interface Props {
  sx?: SxProps;
  iconOnly?: boolean;
  iconSize?: number;
  testId?: string;
}

export default function Logo({
  iconOnly = false,
  iconSize = 16,
  testId,
  sx,
}: Props) {
  return (
    <Box
      component="img"
      data-testid={testId}
      src={iconOnly ? logoIcon : logoText}
      alt="Stormkit Logo"
      sx={{
        maxWidth: "none",
        width: iconSize,
        height: iconOnly ? iconSize : "auto",
        ...sx,
      }}
    />
  );
}
