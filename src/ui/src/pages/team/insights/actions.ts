import { useState, useEffect } from "react";
import api from "~/utils/api/Api";

interface FetchTopDomainsProps {
  teamId?: string;
}

interface TopDomain {
  id: string;
  domainName: string;
  current: number;
  previous: number;
}

export const useFetchTopDomains = ({ teamId }: FetchTopDomainsProps) => {
  const [domains, setTopDomains] = useState<TopDomain[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [paymentRequired, setPaymentRequired] = useState(false);

  useEffect(() => {
    if (!teamId) {
      return;
    }

    setPaymentRequired(false);
    setLoading(true);
    setError("");

    api
      .fetch<{ domains: TopDomain[] }>(`/team/stats/domains?teamId=${teamId}`)
      .then(({ domains }) => {
        setTopDomains(domains);
      })
      .catch(res => {
        if (res.status === 402) {
          setPaymentRequired(true);
          return;
        }

        setError(
          "Something went wrong while fetching domains. Please try again later."
        );
      })
      .finally(() => {
        setLoading(false);
      });
  }, [teamId]);

  return { domains, loading, error, paymentRequired };
};
