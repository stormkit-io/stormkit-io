import type { ButtonProps } from "@mui/material";
import Button from "@mui/material/Button";
import Box from "@mui/material/Box";
import bbLogo from "~/assets/logos/bitbucket-logo.svg";

export default function BitbucketButton(props: ButtonProps) {
  return (
    <Button
      variant="outlined"
      size="large"
      {...props}
      startIcon={<Box component="img" src={bbLogo} alt="Bitbucket" />}
    >
      Bitbucket
    </Button>
  );
}
