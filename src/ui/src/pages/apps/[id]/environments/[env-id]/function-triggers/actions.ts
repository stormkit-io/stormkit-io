import { useEffect, useState } from "react";
import api from "~/utils/api/Api";

interface FetchFunctionTriggersArgs {
  appId?: string;
  environmentId?: string;
  refreshToken?: number;
}

export const useFetchFunctionTriggers = ({
  appId,
  environmentId,
  refreshToken,
}: FetchFunctionTriggersArgs) => {
  const [triggers, setTriggers] = useState<FunctionTrigger[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [paymentRequired, setPaymentRequired] = useState(false);

  useEffect(() => {
    if (!appId || !environmentId) {
      return;
    }

    setLoading(true);

    api
      .fetch<{ triggers: FunctionTrigger[] }>(
        `/v1/triggers?appId=${appId}&envId=${environmentId}`
      )
      .then(({ triggers }) => {
        setTriggers(triggers);
      })
      .catch(res => {
        if (res.status === 402) {
          setPaymentRequired(true);
        } else {
          setError("Something went wrong while fetching periodic triggers");
        }
      })
      .finally(() => {
        setLoading(false);
      });
  }, [appId, environmentId, refreshToken]);

  return { functionTriggers: triggers, loading, error, paymentRequired };
};

interface DeleteFunctionTriggerProps {
  tfid: string;
  appId: string;
  envId: string;
}

export function deleteFunctionTrigger({
  tfid,
  appId,
  envId,
}: DeleteFunctionTriggerProps): Promise<void> {
  return api.delete(
    `/v1/trigger?triggerId=${tfid}&appId=${appId}&envId=${envId}`
  );
}

interface InvokeFunctionTriggerProps {
  tfid: string;
  appId: string;
  envId: string;
}

export function invokeFunctionTrigger({
  tfid,
  appId,
  envId,
}: InvokeFunctionTriggerProps): Promise<void> {
  return api.post(`/v1/trigger/invoke`, {
    id: tfid,
    appId,
    envId,
  });
}

interface CreateFunctionTriggerProps extends Omit<FunctionTrigger, "options"> {
  appId: string;
  envId: string;
  options: FunctionTriggerOptions;
}

export function createFunctionTrigger({
  appId,
  envId,
  cron,
  options,
  status,
}: CreateFunctionTriggerProps): Promise<FunctionTrigger> {
  if (appId === "" || envId === "") {
    return Promise.reject("AppId and EnvId cannot be empty.");
  }

  if (cron === "") {
    return Promise.reject("Cron cannot be empty.");
  }

  if (options.url === "") {
    return Promise.reject("Url cannot be empty.");
  }

  return api.post(`/v1/trigger`, {
    appId,
    envId,
    cron,
    status,
    options,
  });
}

interface UpdateFunctionTriggerProps {
  appId: string;
  envId: string;
  tfid: string;
  cron: string;
  status: boolean;
  options: FunctionTriggerOptions;
}

export const updateFunctionTrigger = ({
  appId,
  envId,
  tfid,
  cron,
  status,
  options,
}: UpdateFunctionTriggerProps): Promise<void> => {
  if (options.url.trim() === "") {
    return Promise.reject("Url cannot be empty.");
  }

  if (cron.trim() === "") {
    return Promise.reject("Cron cannot be empty.");
  }

  return api.patch(`/v1/trigger`, {
    id: tfid,
    appId,
    envId,
    status,
    cron,
    options,
  });
};

interface UpsertFunctionTriggerProps {
  appId: string;
  envId: string;
  tfid?: string;
  status: boolean;
  cron: string;
  options: FunctionTriggerOptions;
}

export const upsertFunctionTrigger = ({
  appId,
  envId,
  tfid,
  status,
  cron,
  options,
}: UpsertFunctionTriggerProps): Promise<void | FunctionTrigger> => {
  if (tfid) {
    return updateFunctionTrigger({ tfid, status, cron, options, appId, envId });
  }

  return createFunctionTrigger({
    appId,
    envId,
    status,
    cron,
    options,
  });
};

interface UseFetchTriggerLogsProps {
  appId: string;
  envId: string;
  triggerId: string;
  refreshToken?: number;
}

export const useFetchTriggerLogs = ({
  appId,
  envId,
  triggerId,
  refreshToken,
}: UseFetchTriggerLogsProps) => {
  const [logs, setLogs] = useState<TriggerLog[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string>();

  useEffect(() => {
    if (!appId || !envId || !triggerId) {
      return;
    }

    setLoading(true);

    api
      .fetch<{ logs: TriggerLog[] }>(
        `/v1/trigger/logs?appId=${appId}&envId=${envId}&triggerId=${triggerId}`
      )
      .then(({ logs }) => {
        setLogs(logs);
      })
      .catch(() => {
        setError("Something went wrong while fetching periodic triggers");
      })
      .finally(() => {
        setLoading(false);
      });
  }, [appId, envId, triggerId, refreshToken]);

  return { logs, loading, error };
};
