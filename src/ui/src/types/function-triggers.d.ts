declare type FunctionTriggerMethod =
  | "POST"
  | "GET"
  | "PUT"
  | "DELETE"
  | "PATCH";

declare interface FunctionTriggerOptions {
  method: FunctionTriggerMethod;
  headers?: Record<string, string>;
  url: string;
  payload?: string;
}

declare interface FunctionTrigger {
  id?: string;
  cron: string;
  /** One-line summary shown next to the trigger in listings. */
  description?: string;
  /** Free-form markdown describing what the trigger is for. */
  documentation?: string;
  status: boolean;
  options: FunctionTriggerOptions;
  nextRunAt?: number;
}

declare interface TriggerLog {
  id?: string;
  triggerId?: string;
  request: {
    headers?: Record<string, string>;
    method?: string;
    url?: string;
    payload?: string;
  };
  response: {
    body?: string;
    code?: number;
    error?: string;
  };
  createdAt: number;
}
