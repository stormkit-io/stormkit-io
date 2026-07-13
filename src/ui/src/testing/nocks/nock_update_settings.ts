import nock from "nock";

const endpoint = process.env.API_DOMAIN || "";

interface UpdateSettingsProps {
  payload: {
    appId: string;
    displayName: string;
    repo: string;
  };
  status?: number;
  response: { ok: boolean };
}

export const mockUpdateSettings = ({
  payload,
  status = 200,
  response = { ok: true },
}: UpdateSettingsProps) =>
  nock(endpoint).put("/app", payload).reply(status, response);
