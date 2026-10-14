import type { Repo } from "../types.d";
import { useEffect, useState } from "react";
import api from "~/utils/api/Api";

const errorMessage =
  "We were not able to fetch Forgejo repositories. " +
  "Please make sure that your Forgejo account is connected.";

interface UseFetchRepoProps {
  page?: number;
  search?: string;
}

export const useFetchRepos = ({ search = "", page = 1 }: UseFetchRepoProps) => {
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

    const params = new URLSearchParams({ search, page: page.toString() });

    api
      .fetch<{ repos: Repo[]; hasNextPage: boolean }>(
        `/provider/forgejo/repos?${params}`
      )
      .then(res => {
        setHasNextPage(res.hasNextPage);
        setRepos(prev => (page > 1 ? [...prev, ...res.repos] : res.repos));
      })
      .catch(() => {
        setError(errorMessage);
      })
      .finally(() => {
        setLoading(false);
        setIsLoadingMore(false);
      });
  }, [page, search]);

  return { repos, hasNextPage, loading, isLoadingMore, error };
};
