import { useEffect, useState } from "react";
import api from "~/utils/api/Api";

interface FetchUniqueVisitorsProps {
  envId: string;
  domainId?: string;
  refreshToken?: number;
  unique?: "true" | "false";
  ts?: "24h" | "7d" | "30d";
}

interface VisitorsChartData {
  name: string;
  total: number;
  unique: number;
}

export const useFetchVisitors = ({
  envId,
  refreshToken,
  domainId,
  unique = "false",
  ts = "24h",
}: FetchUniqueVisitorsProps) => {
  const [visitors, setVisitors] = useState<VisitorsChartData[]>([]);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    // setLoading(true) is not called here because we want to display the spinner only the first time.
    setError("");

    if (!domainId) {
      return;
    }

    api
      .fetch<Record<string, { total: number; unique: number }>>(
        `/analytics/visitors?unique=${unique}&envId=${envId}&ts=${ts}&domainId=${domainId}`
      )
      .then(data => {
        const VisitorsChartData: VisitorsChartData[] = [];

        if (ts === "24h") {
          Object.keys(data).forEach(name => {
            VisitorsChartData.push({
              name,
              total: data[name]?.total || 0,
              unique: data[name]?.unique || 0,
            });
          });
        } else {
          const date = new Date();
          const span = Number(ts.replace("d", ""));
          date.setDate(date.getDate() - span); // Go back 7 or 30 days

          for (let i = 0; i < span; i++) {
            // increment 1 by 1 until today
            date.setDate(date.getDate() + 1);
            const name = date.toISOString().split("T")[0];
            VisitorsChartData.push({
              name,
              total: data[name]?.total || 0,
              unique: data[name]?.unique || 0,
            });
          }
        }

        setVisitors(VisitorsChartData);
      })
      .catch(e => {
        setError("Something went wrong while fetching unique visitors.");
      })
      .finally(() => {
        setLoading(false);
      });
  }, [refreshToken, ts, unique, envId, domainId]);

  return { visitors, error, loading };
};

interface FetchTopReferrerProps {
  envId: string;
  domainId?: string;
  requestPath?: string;
  skip?: boolean;
}

interface Referrer {
  name: string;
  count: number;
}

export const useFetchTopReferrers = ({
  envId,
  domainId,
  requestPath = "",
  skip,
}: FetchTopReferrerProps) => {
  const [referrers, setReferrers] = useState<Referrer[]>([]);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    setReferrers([]);
  }, [requestPath]);

  useEffect(() => {
    if (skip) {
      setLoading(false);
      return;
    }

    if (!domainId) {
      return;
    }

    setLoading(true);

    api
      .fetch<Record<string, number>>(
        `/analytics/referrers?envId=${envId}&domainId=${domainId}&requestPath=${requestPath}`
      )
      .then(data => {
        const refs: Referrer[] = Object.keys(data).map(ref => ({
          name: ref,
          count: data[ref],
        }));

        refs.sort((a, b) =>
          a.count > b.count ? -1 : a.count === b.count ? 0 : 1
        );

        setReferrers(refs);
      })
      .catch(() => {
        setError("Something went wrong while fetching referrers.");
      })
      .finally(() => {
        setLoading(false);
      });
  }, [envId, requestPath, domainId, skip]);

  return { referrers, loading, error };
};

interface FetchTopPathsProps {
  envId: string;
  domainId?: string;
}

interface TopPath {
  name: string;
  count: number;
}

export const useFetchTopPaths = ({ envId, domainId }: FetchTopPathsProps) => {
  const [paths, setPaths] = useState<TopPath[]>([]);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    if (!domainId) {
      return;
    }

    api
      .fetch<Record<string, number>>(
        `/analytics/paths?envId=${envId}&domainId=${domainId}`
      )
      .then(data => {
        const paths: TopPath[] = Object.keys(data).map(ref => ({
          name: ref,
          count: data[ref],
        }));

        paths.sort((a, b) =>
          a.count > b.count ? -1 : a.count === b.count ? 0 : 1
        );

        setPaths(paths);
      })
      .catch(() => {
        setError("Something went wrong while fetching top paths.");
      })
      .finally(() => {
        setLoading(false);
      });
  }, [envId, domainId]);

  return { paths, loading, error };
};

interface FetchEventsProps {
  envId: string;
  domainId?: string;
  ts?: "24h" | "7d" | "30d";
}

interface AnalyticsEvent {
  name: string;
  total: number;
  unique: number;
}

export const useFetchEvents = ({
  envId,
  domainId,
  ts = "30d",
}: FetchEventsProps) => {
  const [events, setEvents] = useState<AnalyticsEvent[]>([]);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    if (!domainId) {
      return;
    }

    setError("");
    setLoading(true);

    api
      .fetch<AnalyticsEvent[]>(
        `/analytics/events?envId=${envId}&domainId=${domainId}&ts=${ts}`
      )
      .then(data => {
        setEvents(Array.isArray(data) ? data : []);
      })
      .catch(() => {
        setError("Something went wrong while fetching events.");
      })
      .finally(() => {
        setLoading(false);
      });
  }, [envId, domainId, ts]);

  return { events, loading, error };
};

interface FetchCountriesProps {
  envId: string;
  domainId?: string;
}

interface ByCountry {
  country: string;
  value: number;
}

export const useFetchByCountries = ({
  envId,
  domainId,
}: FetchCountriesProps) => {
  const [countries, setCountries] = useState<ByCountry[]>([]);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    if (!domainId) {
      return;
    }

    api
      .fetch<Record<string, number>>(
        `/analytics/countries?envId=${envId}&domainId=${domainId}`
      )
      .then(data => {
        const countries: ByCountry[] = Object.keys(data).map(ref => ({
          country: ref,
          value: data[ref],
        }));

        setCountries(countries);
      })
      .catch(() => {
        setError("Something went wrong while fetching countries.");
      })
      .finally(() => {
        setLoading(false);
      });
  }, [envId, domainId]);

  return { countries, loading, error };
};
