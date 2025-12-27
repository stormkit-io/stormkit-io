type autoDeploy = "pull_request" | "commit" | null;

declare interface App {
  id: string;
  teamId: string;
  userId: string;
  repo: string;
  createdAt: number;
  defaultEnv: string;
  defaultEnvId: string;
  displayName: string;
  isBare?: boolean;
}
