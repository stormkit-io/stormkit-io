import { useState, useEffect } from "react";
import api from "~/utils/api/Api";

interface FetchDomainsProps {
  appId: string;
  envId: string;
  search: string;
  afterId?: string;
  refreshToken?: number;
  domainName?: string;
  verified?: boolean;
  enabled?: boolean;
  onFetch?: (d: Domain[]) => void;
}

export const useFetchDomains = ({
  appId,
  envId,
  verified,
  afterId,
  search,
  refreshToken,
  enabled = true,
  onFetch,
}: FetchDomainsProps) => {
  const [domains, setDomains] = useState<Domain[]>([]);
  const [error, setError] = useState<string>();
  const [loading, setLoading] = useState(true);
  const [pagination, setPagination] = useState<Pagination>();
  const [isFirstFetch, setIsFirstFetch] = useState(true);

  useEffect(() => {
    if (!enabled) {
      setLoading(false);
      return;
    }

    const qs = new URLSearchParams(
      JSON.parse(
        JSON.stringify({
          verified,
          envId,
          appId,
          pageSize: 100,
          afterId: search ? undefined : afterId,
          domainName: search || undefined,
        })
      )
    );

    api
      .fetch<{ domains: Domain[]; pagination: Pagination }>(
        `/v1/domains?${qs.toString()}`
      )
      .then(({ domains: newDomains, pagination }) => {
        const allDomains = afterId ? [...domains, ...newDomains] : newDomains;

        // Trigger onFetch only for domain fetches, not for searching
        if (!search && (afterId || isFirstFetch)) {
          onFetch?.(allDomains);
          setIsFirstFetch(false);
        }

        setDomains(allDomains);
        setPagination(pagination);
      })
      .catch(() => {
        setError("Something went wrong while fetching domains.");
      })
      .finally(() => {
        setLoading(false);
      });
  }, [appId, envId, verified, refreshToken, search, afterId, enabled]);

  return { domains, error, loading, pagination };
};
