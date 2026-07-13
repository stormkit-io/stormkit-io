import type { ButtonProps } from "@mui/material";
import Button from "@mui/material/Button";
import Box from "@mui/material/Box";
import githubLogo from "~/assets/logos/github-logo.svg";

export default function GitHubButton(props: ButtonProps) {
  return (
    <Button
      variant="outlined"
      size="large"
      {...props}
      startIcon={<Box component="img" src={githubLogo} alt="GitHub" />}
    >
      GitHub
    </Button>
  );
}
