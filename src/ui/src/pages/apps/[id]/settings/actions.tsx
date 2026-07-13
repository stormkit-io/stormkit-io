import { useEffect, useState } from "react";
import { formatRepo } from "./helpers";
import api from "~/utils/api/Api";
import type { AppSettings } from "./types.d";

interface DeleteAppProps {
  app: App;
}

/**
 * Action to delete an application.
 */
export const deleteApp = ({ app }: DeleteAppProps): Promise<void> => {
  return api.delete("/app", { appId: app.id });
};

export const deleteTrigger = (appId: string) => {
  return api.delete(`/app/${appId}/deploy-trigger`);
};

interface UseFetchAdditionalSettings {
  app: App;
}

interface UseFetchAdditionalSettingsReturnValue {
  loading: boolean;
  error: string | null;
  settings: AppSettings;
  setSettings: (value: React.SetStateAction<AppSettings>) => void;
}

export const useFetchAdditionalSettings = ({
  app,
}: UseFetchAdditionalSettings): UseFetchAdditionalSettingsReturnValue => {
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const [settings, setSettings] = useState<AppSettings>({
    envs: [],
    runtime: "nodejs16.x",
  });

  useEffect(() => {
    let unmounted = false;

    setLoading(true);
    setError(null);
    setSettings({
      envs: [],
      runtime: "nodejs16.x",
    });

    api
      .fetch<AppSettings>(`/app/${app.id}/settings`)
      .then(res => {
        if (unmounted !== true) {
          setSettings({
            deployTrigger: res.deployTrigger,
            runtime: res.runtime,
            envs: res.envs,
          });
        }
      })
      .catch(() => {
        if (unmounted !== true) {
          setError(
            "Failed fetching application settings. Please try again and if the problem persists contact us from Discord or email."
          );
        }
      })
      .finally(() => {
        if (unmounted !== true) {
          setLoading(false);
        }
      });

    return () => {
      unmounted = true;
    };
  }, [app.id, app.refreshToken]);

  return { loading, error, settings, setSettings };
};

interface UpdateAdditionalSettingsProps {
  values: Record<string, string>;
  app: App;
}

export const updateAdditionalSettings = ({
  app,
  values: { repo, displayName, runtime },
}: UpdateAdditionalSettingsProps): Promise<void> => {
  repo = formatRepo(repo);

  return api.put(`/app`, {
    appId: app.id,
    displayName,
    repo,
    runtime,
  });
};
