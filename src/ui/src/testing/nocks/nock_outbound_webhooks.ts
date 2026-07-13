import type { OutboundWebhook } from "~/pages/apps/[id]/settings/types";
import nock from "nock";

const endpoint = process.env.API_DOMAIN || "";

interface FetchOutboundWebhooksProps {
  appId: string;
  status?: number;
  response: { webhooks: OutboundWebhook[] };
}

export const mockFetchOutboundWebhooks = ({
  appId,
  response,
  status = 200,
}: FetchOutboundWebhooksProps) =>
  nock(endpoint).get(`/app/${appId}/outbound-webhooks`).reply(status, response);

interface CreateOutboundWebhookProps {
  appId: string;
  hook: OutboundWebhook;
  status?: number;
}

export const mockCreateOutboundWebhook = ({
  appId,
  hook,
  status = 200,
}: CreateOutboundWebhookProps) =>
  nock(endpoint)
    .post(`/app/outbound-webhooks`, { appId, ...hook })
    .reply(status);

interface UpdateOutboundWebhookProps {
  appId: string;
  whId: string;
  hook: OutboundWebhook;
  status?: number;
}

export const mockUpdateOutboundWebhook = ({
  appId,
  whId,
  hook,
  status = 200,
}: UpdateOutboundWebhookProps) =>
  nock(endpoint)
    .put(`/app/outbound-webhooks`, { appId, whId, ...hook })
    .reply(status);

interface DeleteOutboundWebhookProps {
  appId: string;
  whId: string;
  status?: number;
}

export const mockDeleteOutboundWebhook = ({
  appId,
  whId,
  status = 200,
}: DeleteOutboundWebhookProps) =>
  nock(endpoint)
    .delete(`/app/outbound-webhooks`, { appId, whId })
    .reply(status);
