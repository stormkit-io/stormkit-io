export interface LocationState extends Location {
  triggerDeploysSuccess: string | null;
  integrationsSuccess: string | null;
  outboundWebhooksRefresh: number | null;
  app: number | null;
}

export type Runtime = "nodejs18.x" | "nodejs20.x" | "nodejs22.x";

export interface AppSettings {
  envs: Array<string>;
  deployTrigger?: boolean;
}

export type AllowedMethod = "GET" | "POST" | "HEAD";

export type TriggerWhen =
  | "on_deploy_success"
  | "on_deploy_failed"
  | "on_publish"
  | "on_cache_purge";

export interface OutboundWebhook {
  id?: string;
  triggerWhen: TriggerWhen;
  requestUrl: string;
  requestMethod: AllowedMethod;
  requestPayload?: string;
  requestHeaders?: Record<string, string>;
}

export interface OutboundWebhookFormHeader {
  key: string;
  value: string;
}

export type OutboundWebhookFormValues = Omit<
  OutboundWebhook,
  "requestHeaders"
> & {
  "headers[key]": Array<string> | string;
  "headers[value]": Array<string> | string;
};
