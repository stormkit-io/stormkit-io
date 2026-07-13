import type { Repo } from "../types.d";
import { useEffect, useState } from "react";
import gitlabApi from "~/utils/api/Gitlab";

const errorMessage =
  "We were not able to fetch GitLab repositories. " +
  "Please make sure that we have enough permissions.";

interface UseFetchRepoProps {
  page?: number;
}

interface UseFetchReposReturnValue {
  loading: boolean;
  isLoadingMore: boolean;
  error?: string;
  repos: Repo[];
  hasNextPage: boolean;
}

export const useFetchRepos = ({
  page = 1,
}: UseFetchRepoProps): UseFetchReposReturnValue => {
  const [loading, setLoading] = useState(true);
  const [isLoadingMore, setIsLoadingMore] = useState(false);
  const [error, setError] = useState<string>();
  const [repos, setRepos] = useState<Repo[]>([]);
  const [hasNextPage, setHasNextPage] = useState(false);

  useEffect(() => {
    if (page > 1) {
      setIsLoadingMore(true);
    } else {
      setLoading(true);
    }

    gitlabApi
      .repositories({
        page,
      })
      .then(res => {
        setHasNextPage(res.nextPage != "");

        let repoList: Repo[] = [];

        if (page > 1) {
          repoList = repos;
        }

        setRepos([
          ...repoList,
          ...res.repos.map(r => ({
            name: r.path_with_namespace,
            fullName: r.path_with_namespace,
          })),
        ]);
      })
      .catch(() => {
        setError(errorMessage);
      })
      .finally(() => {
        setLoading(false);
        setIsLoadingMore(false);
      });
  }, [page]);

  return { repos, hasNextPage, loading, isLoadingMore, error };
};
