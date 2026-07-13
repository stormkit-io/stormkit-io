import type { OutboundWebhook } from "~/pages/apps/[id]/settings/types";

export default (): OutboundWebhook[] => [
  {
    requestUrl: "https://discord.com/example/endpoint",
    requestMethod: "POST",
    requestPayload: '{"embeds":[]}',
    requestHeaders: { "content-type": "application/json" },
    triggerWhen: "on_publish",
    id: "2",
  },
  {
    requestUrl:
      "https://discord.com/api/webhooks/example/endpoint/with/a/very/long/url/structure",
    requestMethod: "POST",
    requestPayload: '{"embeds":[]}',
    requestHeaders: { "content-type": "application/json" },
    triggerWhen: "on_publish",
    id: "3289604703",
  },
];
