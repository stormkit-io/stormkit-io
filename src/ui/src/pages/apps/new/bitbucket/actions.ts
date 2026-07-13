import type { Repo } from "../types.d";
import { useEffect, useState } from "react";
import { useSearchParams } from "react-router-dom";
import bitbucketApi from "~/utils/api/Bitbucket";

const errorMessage =
  "We were not able to fetch Bitbucket repositories. " +
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
  const [searchParams, _] = useSearchParams();
  const role = searchParams.get("role");

  useEffect(() => {
    if (page > 1) {
      setIsLoadingMore(true);
    } else {
      setLoading(true);
    }

    bitbucketApi
      .repositories({
        params: {
          role: role === "member" ? role : "admin",
          pagelen: 100,
        },
      })
      .then(res => {
        setHasNextPage(Boolean(res.next));

        let repoList: Repo[] = [];

        if (page > 1) {
          repoList = repos;
        }

        setRepos([
          ...repoList,
          ...res.values
            .filter(r => r.type === "repository")
            .map(r => ({
              name: r.full_name,
              fullName: r.full_name,
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
  }, [page, role]);

  return { repos, hasNextPage, loading, isLoadingMore, error };
};
