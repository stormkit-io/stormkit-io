import { useState, useEffect, useContext } from "react";
import { useSearchParams } from "react-router-dom";
import Box from "@mui/material/Box";
import Link from "@mui/material/Link";
import CheckIcon from "@mui/icons-material/Check";
import Card from "~/components/Card";
import CardHeader from "~/components/CardHeader";
import { AuthContext } from "~/pages/auth/Auth.context";
import * as buttons from "~/components/Buttons";
import api from "~/utils/api/Api";
import GitHubModal from "./GitHubModal";
import GitLabModal from "./GitLabModal";
import BitbucketModal from "./BitbucketModal";
import { GitDetails } from "./types";

const { GithubButton, GitlabButton, BitbucketButton } = buttons;

const useFetchGitDetails = () => {
  const [loading, setLoading] = useState(true);
  const [details, setDetails] = useState<GitDetails>();
  const [error, setError] = useState<string>();

  useEffect(() => {
    api
      .fetch<GitDetails>("/admin/git/details")
      .then(details => {
        setDetails(details);
      })
      .catch(() => {
        setError("Something went wrong while fetching git details.");
      })
      .finally(() => {
        setLoading(false);
      });
  }, []);

  return { details, loading, error };
};

export default function Git() {
  const { logout } = useContext(AuthContext);
  const [params, _] = useSearchParams();
  const [isGitHubModalOpen, setIsGitHubModalOpen] = useState(false);
  const [isGitLabModalOpen, setIsGitLabModalOpen] = useState(false);
  const [isBitbucketModalOpen, setIsBitbucketModalOpen] = useState(false);
  const [error, setError] = useState<string>();
  const [success, setSuccess] = useState<string>();
  const { details, loading, error: fetchError } = useFetchGitDetails();
  const isWebhookSecretMissing =
    details?.github && !details.github.hasWebhookSecret;

  useEffect(() => {
    if (params.get("success") === "github_app_created") {
      logout?.();
    }

    if (params.get("error") === "github_app_state_invalid") {
      setError(
        "The GitHub App setup link expired or was already used, so the app was not configured. Delete the app on GitHub if it was created, then start again."
      );
    }
  }, [params]);

  return (
    <Card
      error={error || fetchError}
      success={success}
      info={
        isWebhookSecretMissing && (
          <>
            GitHub webhooks are rejected because no webhook secret is
            configured, so pushes and pull requests do not trigger deployments.{" "}
            <Link
              component="button"
              color="secondary"
              onClick={() => setIsGitHubModalOpen(true)}
            >
              Configure the webhook secret
            </Link>{" "}
            of your GitHub App to enable automatic deployments.
          </>
        )
      }
      loading={loading}
      sx={{ backgroundColor: "container.transparent" }}
    >
      <CardHeader
        title="Git Provider Configuration"
        subtitle="Configure authentication for your Git providers"
      />
      <Box
        sx={{
          display: "flex",
          mb: error || success || isWebhookSecretMissing ? 4 : 0,
          mx: "auto",
        }}
      >
        <GithubButton
          sx={{ mr: 2 }}
          onClick={() => setIsGitHubModalOpen(true)}
          endIcon={
            details?.github ? (
              <CheckIcon color="success" sx={{ scale: 0.7 }} />
            ) : null
          }
        >
          Configure GitHub
        </GithubButton>
        <GitlabButton
          sx={{ mr: 2 }}
          onClick={() => setIsGitLabModalOpen(true)}
          endIcon={
            details?.gitlab ? (
              <CheckIcon color="success" sx={{ scale: 0.7 }} />
            ) : null
          }
        >
          Configure GitLab
        </GitlabButton>
        <BitbucketButton
          onClick={() => setIsBitbucketModalOpen(true)}
          endIcon={
            details?.bitbucket ? (
              <CheckIcon color="success" sx={{ scale: 0.7 }} />
            ) : null
          }
        >
          Configure Bitbucket
        </BitbucketButton>
      </Box>

      {isGitHubModalOpen && (
        <GitHubModal
          details={details}
          closeModal={() => setIsGitHubModalOpen(false)}
          onSuccess={(message: string) => {
            setSuccess(message);
            setError(undefined);
            setIsGitHubModalOpen(false);
            setTimeout(() => window.location.reload(), 2000);
          }}
        />
      )}

      {isGitLabModalOpen && (
        <GitLabModal
          details={details}
          closeModal={() => setIsGitLabModalOpen(false)}
          onSuccess={(message: string) => {
            setSuccess(message);
            setError(undefined);
            setIsGitLabModalOpen(false);
            setTimeout(() => window.location.reload(), 2000);
          }}
        />
      )}

      {isBitbucketModalOpen && (
        <BitbucketModal
          details={details}
          closeModal={() => setIsBitbucketModalOpen(false)}
          onSuccess={(message: string) => {
            setSuccess(message);
            setError(undefined);
            setIsBitbucketModalOpen(false);
            setTimeout(() => window.location.reload(), 2000);
          }}
        />
      )}
    </Card>
  );
}
