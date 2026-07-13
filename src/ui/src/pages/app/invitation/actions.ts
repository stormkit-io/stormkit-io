import { useEffect, useState } from "react";
import { useLocation } from "react-router-dom";
import qs from "query-string";
import api from "~/utils/api/Api";

interface EnrollResponse {
  appId: string;
}

export const useParseAppId = (): string => {
  const location = useLocation();
  const { token } = qs.parse(location.search);
  const [appId, setAppId] = useState<string>("");

  useEffect(() => {
    let unmounted = false;

    api.post<EnrollResponse>("/app/members/enroll", { token }).then(res => {
      if (!unmounted) {
        setAppId(res.appId);
      }
    });

    return () => {
      unmounted = true;
    };
  }, []);

  return appId;
};
