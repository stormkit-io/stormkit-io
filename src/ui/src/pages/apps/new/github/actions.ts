import type { Repo, Account } from "../types.d";
import { useEffect, useState } from "react";
import api from "~/utils/api/Api";

const errorMessage =
  "We were not able to fetch GitHub repositories. " +
  "Please make sure that we have enough permissions.";

interface UseFetchRepoProps {
  page?: number;
  search?: string;
  installationId?: string;
}

export const useFetchRepos = ({
  installationId,
  search = "",
  page = 1,
}: UseFetchRepoProps) => {
  const [loading, setLoading] = useState<boolean | null>(null);
  const [isLoadingMore, setIsLoadingMore] = useState(true);
  const [error, setError] = useState<string>();
  const [repos, setRepos] = useState<Repo[]>([]);
  const [hasNextPage, setHasNextPage] = useState<boolean>();

  useEffect(() => {
    if (!installationId) {
      return;
    }

    if (page > 1) {
      setIsLoadingMore(true);
    } else {
      setLoading(true);
    }

    api
      .fetch<{ repos: Repo[]; hasNextPage: boolean }>(
        `/provider/github/repos?search=${search}&page=${page}&installationId=${installationId}`
      )
      .then(res => {
        setHasNextPage(res.hasNextPage);

        if (search) {
          setRepos(res.repos);
        } else {
          setRepos([...repos, ...res.repos]);
        }
      })
      .catch(() => {
        setError(errorMessage);
      })
      .finally(() => {
        setLoading(false);
        setIsLoadingMore(false);
      });
  }, [page, installationId, search]);

  return {
    repos,
    setRepos,
    hasNextPage: Boolean(hasNextPage),
    loading,
    isLoadingMore,
    error,
  };
};

interface FetchAccountProps {
  refreshToken?: number;
}

export const useFetchAccounts = ({ refreshToken }: FetchAccountProps) => {
  const [error, setError] = useState<string>();
  const [loading, setLoading] = useState(true);
  const [accounts, setAccounts] = useState<Account[]>([]);
  const [needsAuth, setNeedsAuth] = useState(false);

  useEffect(() => {
    setLoading(true);

    api
      .fetch<{ accounts: Account[] }>("/provider/github/accounts")
      .then(({ accounts }) => {
        setAccounts(accounts);
      })
      .catch(res => {
        if (res.status === 401) {
          setNeedsAuth(true);
          return;
        }

        setError(errorMessage);
      })
      .finally(() => {
        setLoading(false);
      });
  }, [refreshToken]);

  return { error, loading, accounts, needsAuth };
};
