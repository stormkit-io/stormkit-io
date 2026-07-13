import nock from "nock";

const endpoint = process.env.API_DOMAIN || "";

interface MockFetchRecordsProps {
  appId: string;
  envId: string;
  collectionName: string;
  status?: number;
  response: CollectionRecord[];
}

export const mockFetchRecords = ({
  appId,
  envId,
  collectionName,
  status = 200,
  response = [],
}: MockFetchRecordsProps) => {
  return nock(endpoint)
    .post(`/app/data-storage/query`, { appId, envId, collectionName })
    .reply(status, response);
};

interface MockFetchCollectionsProps {
  appId: string;
  envId: string;
  status?: number;
  response: Collection[];
}

export const mockFetchCollections = ({
  appId,
  envId,
  status = 200,
  response = [],
}: MockFetchCollectionsProps) => {
  return nock(endpoint)
    .post(`/app/data-storage/collections`, { appId, envId })
    .reply(status, response);
};
