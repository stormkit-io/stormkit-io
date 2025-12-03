import { useState, useContext, useEffect, useMemo } from "react";
import { AuthContext } from "~/pages/auth/Auth.context";
import { RootContext } from "~/pages/Root.context";
import Box from "@mui/material/Box";
import Link from "@mui/material/Link";
import Typography from "@mui/material/Typography";
import ArrowBack from "@mui/icons-material/ArrowBack";
import Button from "@mui/material/Button";
import Card from "~/components/Card";
import CardHeader from "~/components/CardHeader";
import RepoList from "../_components/RepoList";
import Accounts from "../_components/Accounts";
import { useFetchAccounts, useFetchRepos } from "./actions";
import ConnectMoreRepos from "./ConnectMoreRepos";

function NeedsAuth() {
  const { loginOauth } = useContext(AuthContext);

  return (
    <Typography
      sx={{
        display: "flex",
        justifyContent: "space-between",
        alignItems: "center",
        gap: 1,
      }}
    >
      <Box sx={{ flexGrow: 1 }} component="span">
        You need to authenticate to proceed.
      </Box>
      <Button
        variant="contained"
        color="secondary"
        type="button"
        onClick={() => loginOauth?.("github")}
      >
        Authenticate
      </Button>
    </Typography>
  );
}

export default function NewGithubApp() {
  const { user } = useContext(AuthContext);
  const { details, loading: detailsLoading } = useContext(RootContext);
  const { githubAccount, openPopupURL } = useMemo(() => {
    const githubAccount = details?.auth?.github;
    const openPopupURL = `https://github.com/apps/${githubAccount}/installations/new`;
    return { githubAccount, openPopupURL };
  }, [details?.auth, detailsLoading]);
  const [page, setPage] = useState(1);
  const [search, setSearch] = useState<string>();
  const [refreshToken, setRefreshToken] = useState<number>();
  const [installationId, setInstallationId] = useState<string>();
  const {
    accounts,
    error: faError,
    loading: faLoading,
    needsAuth,
  } = useFetchAccounts({ refreshToken });

  const {
    repos,
    setRepos,
    hasNextPage,
    error: frError,
    loading: frLoading,
    isLoadingMore,
  } = useFetchRepos({ installationId, search, page });

  useEffect(() => {
    for (let i = 0; i < accounts.length; i++) {
      if (accounts[i]?.login === user?.displayName) {
        setInstallationId(accounts[i].id);
        break;
      }

      // If there is no match between current user display name and
      // the accounts, then select the first one.
      // Why we have this?
      // If you're invited as a member to other repos, and they are also
      // using stormkit, github might return other accounts first. So instead
      // of seeing repos from your individual account first, you see other people's repositories.
      // To prevent this, above we check that displayName === account.login. If nothing matches we
      // select the first one.
      if (accounts[0]) {
        setInstallationId(accounts[0].id);
      }
    }
  }, [accounts, user?.displayName]);

  const hasAnyAccount = accounts && accounts.length > 0;
  const showNoRepo = frLoading === false && !repos?.length;

  return (
    <Box maxWidth="md" sx={{ width: "100%", mx: "auto", mt: 2 }}>
      <Card
        sx={{ width: "100%", mb: 4 }}
        loading={faLoading}
        info={
          needsAuth ? (
            <NeedsAuth />
          ) : (
            showNoRepo && (
              <Typography>
                No repositories found. Try connecting more repositories.
              </Typography>
            )
          )
        }
        error={!hasAnyAccount ? undefined : faError || frError}
      >
        <CardHeader
          actions={
            !faLoading &&
            !needsAuth && (
              <ConnectMoreRepos
                setInstallationId={setInstallationId}
                setRefreshToken={setRefreshToken}
                openPopupURL={openPopupURL}
              />
            )
          }
        >
          <Typography>
            <Link
              sx={{ display: "inline-flex", alignItems: "center" }}
              href="/"
            >
              <ArrowBack sx={{ mr: 1 }} />
              Import from GitHub
            </Link>
          </Typography>
        </CardHeader>
        <Box>
          {!needsAuth && accounts?.length > 0 && (
            <Box>
              <Accounts
                accounts={accounts}
                selected={installationId}
                onAccountChange={id => {
                  setPage(1);
                  setRepos([]);
                  setInstallationId(id);
                }}
              />
            </Box>
          )}

          {!needsAuth && (
            <Box
              sx={{
                mb: !githubAccount ? 4 : 0,
              }}
            >
              <RepoList
                repositories={repos}
                provider="github"
                isLoadingList={frLoading === true}
                isLoadingMore={isLoadingMore}
                hasNextPage={hasNextPage}
                onNextPage={() => setPage(page + 1)}
                onSearch={term => {
                  setSearch(term);
                  setRepos([]);
                }}
              />
            </Box>
          )}
        </Box>
      </Card>
    </Box>
  );
}
