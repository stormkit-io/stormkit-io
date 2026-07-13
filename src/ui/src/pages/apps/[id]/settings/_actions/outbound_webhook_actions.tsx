import { useEffect, useState } from "react";
import api from "~/utils/api/Api";
import type { OutboundWebhook } from "../types";

interface FetchOutboundWebhooksProps {
  app: App;
  refreshToken?: number;
}

interface FetchOutboundWebhooksReturnValue {
  hooks: Array<OutboundWebhook>;
  error?: string;
  loading: boolean;
}

interface FetchOutboundWebhooksRequest {
  webhooks: Array<OutboundWebhook>;
}

export const useFetchOutboundWebhooks = ({
  app,
  refreshToken,
}: FetchOutboundWebhooksProps): FetchOutboundWebhooksReturnValue => {
  const [hooks, setHooks] = useState<Array<OutboundWebhook>>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string>();

  useEffect(() => {
    let unmounted = false;

    setError(undefined);
    setLoading(true);

    api
      .fetch<FetchOutboundWebhooksRequest>(`/app/${app.id}/outbound-webhooks`)
      .then(res => {
        if (unmounted !== true) {
          setHooks(res.webhooks);
        }
      })
      .catch(() => {
        if (unmounted !== true) {
          setError(
            "Something went wrong while fetching the outbound webhooks."
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
  }, [app.id, refreshToken]);

  return { hooks, error, loading };
};

interface UpsertOutboundWebhookProps extends OutboundWebhook {
  app: App;
}

export const upsertOutboundWebhook = ({
  app,
  id,
  requestUrl,
  requestHeaders,
  requestMethod,
  requestPayload,
  triggerWhen,
}: UpsertOutboundWebhookProps): Promise<void> => {
  const hooks: OutboundWebhook = {
    requestUrl,
    requestMethod,
    requestPayload,
    triggerWhen,
    requestHeaders,
  };

  const method = id ? "put" : "post";

  return api[method](`/app/outbound-webhooks`, {
    ...hooks,
    whId: id || undefined,
    appId: app.id,
  });
};

interface SendSampleRequestProps {
  app: App;
  whId?: string;
}

export interface SendSampleRequestResponse {
  error?: string;
  result: {
    status: number;
    body: string;
  };
}

export const sendSampleRequest = ({
  app,
  whId,
}: SendSampleRequestProps): Promise<SendSampleRequestResponse> => {
  return api.fetch<SendSampleRequestResponse>(
    `/app/${app.id}/outbound-webhooks/${whId}/trigger`
  );
};

interface DeleteOutboundWebhookProps {
  app: App;
  whId?: string;
}

export const deleteOutboundWebhook = ({
  app,
  whId,
}: DeleteOutboundWebhookProps): Promise<void> => {
  return api.delete(`/app/outbound-webhooks`, {
    whId,
    appId: app.id,
  });
};
