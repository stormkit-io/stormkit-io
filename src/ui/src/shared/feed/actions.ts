import { useEffect, useState } from "react";
import api from "~/utils/api/Api";

interface UseFetchAuditsProps {
  appId?: string;
  envId?: string;
  teamId?: string;
  nextPage?: number;
}

export const useFetchAudits = ({
  teamId,
  envId,
  appId,
  nextPage,
}: UseFetchAuditsProps) => {
  const [audits, setAudits] = useState<Audit[]>([]);
  const [error, setError] = useState<string>();
  const [loading, setLoading] = useState(true);
  const [beforeId, setBeforeId] = useState("");
  const [paymentRequired, setPaymentRequired] = useState(false);

  const params = new URLSearchParams(
    JSON.parse(
      JSON.stringify({
        teamId,
        appId,
        envId,
        beforeId,
      })
    )
  );

  useEffect(() => {
    if (!appId && !teamId && !envId) {
      setLoading(false);
      return;
    }

    setLoading(true);
    setError(undefined);
    setPaymentRequired(false);

    api
      .fetch<{ audits: Audit[]; pagination: Pagination }>(
        `/audits?${params.toString()}`
      )
      .then(({ audits: newAudits, pagination }) => {
        setAudits(beforeId ? [...audits, ...newAudits] : newAudits);

        if (pagination.hasNextPage && pagination.beforeId) {
          setBeforeId(pagination.beforeId);
        } else {
          setBeforeId("");
        }
      })
      .catch(res => {
        if (res.status === 402) {
          setPaymentRequired(true);
          return;
        }

        setError(
          "Something went wrong while fetching audits. Try again later."
        );
      })
      .finally(() => {
        setLoading(false);
      });
  }, [envId, appId, teamId, nextPage]);

  return {
    audits,
    error,
    loading,
    paymentRequired,
    hasNextPage: beforeId !== "",
  };
};
