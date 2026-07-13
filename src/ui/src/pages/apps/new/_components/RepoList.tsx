import type { Repo } from "../types.d";
import { useState, useMemo, useContext } from "react";
import { useNavigate } from "react-router";
import Box from "@mui/material/Box";
import TextField from "@mui/material/TextField";
import Button from "@mui/material/Button";
import SearchIcon from "@mui/icons-material/Search";
import Typography from "@mui/material/Typography";
import ArrowRightIcon from "@mui/icons-material/ArrowForwardIos";
import { useSelectedTeam } from "~/layouts/TopMenu/Teams/actions";
import { AuthContext } from "~/pages/auth/Auth.context";
import githubLogo from "~/assets/logos/github-logo.svg";
import bitbucketLogo from "~/assets/logos/bitbucket-logo.svg";
import gitlabLogo from "~/assets/logos/gitlab-logo.svg";
import { insertRepo } from "./actions";
import { CircularProgress } from "@mui/material";

const logos: Record<Provider, string> = {
  github: githubLogo,
  bitbucket: bitbucketLogo,
  gitlab: gitlabLogo,
};

export interface Props {
  repositories: Repo[];
  provider: Provider;
  isLoadingList: boolean;
  isLoadingMore: boolean;
  hasNextPage: boolean;
  onNextPage: () => void;
  onSearch?: (term: string) => void;
}

let filterTimer: NodeJS.Timeout;

export default function RepoList({
  repositories,
  provider,
  isLoadingList,
  isLoadingMore,
  hasNextPage,
  onNextPage,
  onSearch,
}: Props) {
  const [filter, setFilter] = useState<string>("");
  const [loadingInsert, setLoadingInsert] = useState("");
  const { teams } = useContext(AuthContext);
  const team = useSelectedTeam({ teams });
  const navigate = useNavigate();
  const logo = logos[provider];
  const repos = useMemo<Repo[]>(
    () => repositories.filter(r => !filter || r.name.includes(filter)),
    [repositories, filter]
  );

  const handleRepoInsert = (r: Repo) => {
    setLoadingInsert(r.fullName);
    insertRepo({
      repo: r.fullName,
      provider,
      teamId: team?.id,
    })
      .then(app => {
        navigate(`/apps/${app.id}`);
      })
      .finally(() => {
        setLoadingInsert("");
      });
  };

  return (
    <>
      <TextField
        fullWidth
        label={onSearch ? "Search repositories" : "Filter repositories"}
        variant="filled"
        placeholder="Search repositories by name"
        sx={{ mb: 2 }}
        autoFocus
        onChange={e => {
          clearTimeout(filterTimer);
          filterTimer = setTimeout(() => {
            if (onSearch) {
              onSearch(e.target.value);
            } else {
              setFilter(e.target.value);
            }
          }, 250);
        }}
        slotProps={{
          inputLabel: { sx: { pl: 1 } },
          input: {
            sx: {
              pl: 0.75,
            },
            endAdornment: isLoadingList ? (
              <CircularProgress size="1rem" />
            ) : (
              <SearchIcon sx={{ width: "1rem" }} />
            ),
          },
        }}
      />

      {repos.map(r => (
        <Box
          key={r.fullName}
          sx={{
            display: "flex",
            alignItems: "center",
            px: 2,
            pr: 1,
            py: 1,
            borderBottom: `1px solid`,
            borderColor: "container.border",
            ":hover": {
              bgcolor: "container.transparent",
            },
            ":last-child": {
              borderBottom: "none",
            },
          }}
          aria-label={r.name}
          tabIndex={0}
          role="button"
          onKeyUp={e => {
            if (e.key === "Enter") {
              handleRepoInsert(r);
            }
          }}
          onClick={() => {
            handleRepoInsert(r);
          }}
        >
          <Box
            component="img"
            src={logo}
            alt={provider}
            sx={{ width: 24, mr: 1.5 }}
          />
          <Typography component="span" sx={{ flex: 1 }}>
            {r.name}
          </Typography>
          <Box>
            <Button
              variant="text"
              color="info"
              sx={{
                textTransform: "uppercase",
                fontSize: 12,
              }}
              endIcon={<ArrowRightIcon sx={{ scale: "0.75" }} />}
              loading={loadingInsert === r.fullName}
            >
              import
            </Button>
          </Box>
        </Box>
      ))}
      {hasNextPage && repos.length > 0 && (
        <Box sx={{ mt: 2, textAlign: "center" }}>
          <Button
            variant="contained"
            color="secondary"
            loading={isLoadingMore}
            onClick={onNextPage}
          >
            Load more
          </Button>
        </Box>
      )}
    </>
  );
}
