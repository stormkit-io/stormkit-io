import nock from "nock";

const endpoint = process.env.API_DOMAIN || "";

export interface MockMailerConfig {
  host: string;
  port: string;
  username: string;
  password: string;
  useDefault?: boolean;
}

interface MockFetchMailerConfigProps {
  appId?: string;
  envId?: string;
  teamId?: string;
  status?: number;
  response?: { config: MockMailerConfig; defaultAvailable?: boolean };
}

export const mockFetchMailerConfig = ({
  appId = "",
  envId = "",
  status = 200,
  response,
}: MockFetchMailerConfigProps) => {
  return nock(endpoint)
    .get(`/v1/mailer/config?appId=${appId}&envId=${envId}`)
    .reply(status, response);
};

interface MockSetMailerConfigProps {
  appId?: string;
  envId?: string;
  smtpHost?: string;
  smtpPort?: string;
  username?: string;
  password?: string;
  useDefault?: boolean;
  status?: number;
  response?: { ok: boolean } | { error: string };
}

export const mockSetMailerConfig = ({
  appId,
  envId,
  smtpHost,
  smtpPort,
  username,
  password,
  useDefault = false,
  status = 200,
  response = { ok: true },
}: MockSetMailerConfigProps) => {
  const body = useDefault
    ? { appId, envId, useDefault }
    : { appId, envId, smtpHost, smtpPort, username, password, useDefault };

  return nock(endpoint).post(`/v1/mailer/config`, body).reply(status, response);
};

export interface MockEmail {
  id: string;
  envId: string;
  from: string;
  to: string;
  subject: string;
  body: string;
  sentAt: number;
}

interface MockFetchEmailsProps {
  appId?: string;
  envId?: string;
  status?: number;
  response?: { emails: MockEmail[] };
}

export const mockFetchEmails = ({
  appId = "",
  envId = "",
  status = 200,
  response = { emails: [] },
}: MockFetchEmailsProps) => {
  return nock(endpoint)
    .get(`/mailer?appId=${appId}&envId=${envId}`)
    .reply(status, response);
};

interface MockSendTestEmailProps {
  appId: string;
  envId: string;
  from: string;
  to: string;
  delivered?: boolean;
}

export const mockSendTestEmail = ({
  appId,
  envId,
  from,
  to,
  delivered = true,
}: MockSendTestEmailProps) => {
  return nock(endpoint)
    .post(
      "/v1/mail",
      JSON.stringify({
        appId,
        envId,
        from,
        to,
        body: "Test email body",
        subject: "Test email subject",
      }),
    )
    .reply(200, { ok: true, delivered });
};
