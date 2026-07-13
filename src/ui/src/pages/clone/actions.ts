import { useEffect, useState } from "react";
import api from "~/utils/api/Api";

export const useClone = (
  template: string | (string | null)[] | null
): { loading: boolean; error: string | null } => {
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  if (template == null) {
    setError("Can't find repo name");
    setLoading(false);
    return { loading, error };
  }

  useEffect(() => {
    let unmounted = false;

    setLoading(true);
    setError(null);

    api
      .fetch(`/deploy?template=${template}`)
      .then(_ => {
        if (unmounted !== true) {
          setLoading(false);
        }
      })
      .catch(async e => {
        if (unmounted !== true) {
          const errorBody= await e.json();
          setError(errorBody.message);
          setLoading(false);
        }
      });

    return () => {
      unmounted = true;
    };
  }, [api]);

  return { loading, error };
};
