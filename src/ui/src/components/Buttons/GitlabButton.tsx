import type { ButtonProps } from "@mui/material";
import Button from "@mui/material/Button";
import Box from "@mui/material/Box";
import gitlabLogo from "~/assets/logos/gitlab-logo.svg";

export default function GitLabButton(props: ButtonProps) {
  return (
    <Button
      variant="outlined"
      size="large"
      {...props}
      startIcon={<Box component="img" src={gitlabLogo} alt="GitLab" />}
    >
      GitLab
    </Button>
  );
}
