import { useContext, useState } from "react";
import { useNavigate } from "react-router";
import { useSearchParams } from "react-router-dom";
import Box from "@mui/material/Box";
import LoadingButton from "@mui/lab/LoadingButton";
import CardFooter from "~/components/CardFooter";
import CardHeader from "~/components/CardHeader";
import Card from "~/components/Card";
import TextField from "@mui/material/TextField";
import Link from "@mui/material/Link";
import Typography from "@mui/material/Typography";
import ArrowBack from "@mui/icons-material/ArrowBack";
import { useSelectedTeam } from "~/layouts/TopMenu/Teams/actions";
import { AuthContext } from "~/pages/auth/Auth.context";
import { insertRepo } from "../_components/actions";

export default function ImportURL() {
  const [searchParams] = useSearchParams();
  const { teams } = useContext(AuthContext);
  const team = useSelectedTeam({ teams });
  const defaultURL = searchParams.get("r") || "";
  const [importUrl, setImportUrl] = useState<string>(defaultURL);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string>();
  const navigate = useNavigate();

  const handleRepoInsert = () => {
    let provider: Provider | undefined;
    let repoName: string | undefined;

    try {
      const parsed = new URL(importUrl);
      repoName = parsed.pathname.replace(/^\/+/, "").replace(".git", "");
      provider = parsed.hostname.includes("github")
        ? "github"
        : parsed.hostname.includes("gitlab")
        ? "gitlab"
        : parsed.hostname.includes("bitbucket")
        ? "bitbucket"
        : undefined;
    } catch {}

    if (!provider || !repoName) {
      return setError(
        "Make sure to import the repository from one these providers: GitLab, GitHub, Bitbucket."
      );
    }

    setIsLoading(true);
    setError(undefined);

    insertRepo({
      provider: "github",
      repo: repoName,
      teamId: team?.id,
    })
      .then(app => {
        navigate(`/apps/${app.id}/environments`);
      })
      .catch(() => {
        setError("Something went wrong while importing the repository.");
      })
      .finally(() => {
        setIsLoading(false);
      });
  };

  return (
    <Card maxWidth="md" error={error} sx={{ mt: 2 }}>
      <CardHeader>
        <Typography variant="h2">
          <Link href="/" sx={{ display: "inline-flex", alignItems: "center" }}>
            <ArrowBack sx={{ mr: 1 }} />
            Import from URL
          </Link>
        </Typography>
      </CardHeader>
      <Box sx={{ mb: 4 }}>
        <TextField
          color="info"
          variant="filled"
          label="Where can we find the repository?"
          placeholder="https://github.com/stormkit-io/marketplace-feedback-form"
          fullWidth
          autoComplete="off"
          value={importUrl}
          onChange={e => setImportUrl(e.currentTarget.value)}
        />
      </Box>
      <CardFooter>
        <LoadingButton
          loading={isLoading}
          variant="contained"
          color="secondary"
          onClick={handleRepoInsert}
        >
          Import repo
        </LoadingButton>
      </CardFooter>
    </Card>
  );
}
