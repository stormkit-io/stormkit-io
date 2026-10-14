import type { ButtonProps } from "@mui/material";
import Button from "@mui/material/Button";
import Box from "@mui/material/Box";
import forgejoLogo from "~/assets/logos/forgejo-logo.svg";

export default function ForgejoButton(props: ButtonProps) {
  return (
    <Button
      variant="outlined"
      size="large"
      {...props}
      startIcon={<Box component="img" src={forgejoLogo} alt="Forgejo" />}
    >
      Forgejo
    </Button>
  );
}
