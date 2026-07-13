import nock from "nock";

const endpoint = process.env.API_DOMAIN || "";

interface MockFetchVolumesConfigProps {
  response: { config: boolean };
  status: number;
}

export const mockFetchVolumesConfig = ({
  status = 200,
  response,
}: MockFetchVolumesConfigProps) =>
  nock(endpoint).get("/volumes/config").reply(status, response);

interface MockSetVolumesConfig {
  status?: number;
  response?: { ok: boolean };
  config: VolumeConfig;
}

export const mockSetVolumesConfig = ({
  config,
  status = 200,
  response = { ok: true },
}: MockSetVolumesConfig) =>
  nock(endpoint)
    .post("/volumes/config", { ...config })
    .reply(status, response);

interface MockFetchFilesProps {
  appId: string;
  envId: string;
  beforeId?: string;
  status?: number;
  response?: { files: VolumeFile[] };
}

export const mockFetchFiles = ({
  appId,
  envId,
  beforeId,
  status = 200,
  response = { files: [] },
}: MockFetchFilesProps) =>
  nock(endpoint)
    .get(`/volumes?appId=${appId}&envId=${envId}&beforeId=${beforeId}`)
    .reply(status, response);

interface RemoveFilesProps {
  appId: string;
  envId: string;
  fileId: string;
  status?: number;
  response?: { ok: boolean };
}

export const mockRemoveFiles = ({
  appId,
  envId,
  fileId,
  status = 200,
  response = { ok: true },
}: RemoveFilesProps) =>
  nock(endpoint)
    .delete(`/volumes?appId=${appId}&envId=${envId}&ids=${fileId}`)
    .reply(status, response);

interface ToggleVisibilityProps {
  appId: string;
  envId: string;
  fileId: string;
  visibility: "public" | "private";
  status?: number;
  response?: { ok: boolean };
}

export const mockToggleVisibility = ({
  appId,
  envId,
  fileId,
  visibility,
  status = 200,
  response = { ok: true },
}: ToggleVisibilityProps) =>
  nock(endpoint)
    .post("/volumes/visibility", { appId, envId, fileId, visibility })
    .reply(status, response);
