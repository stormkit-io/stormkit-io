declare interface APIKey {
  id: string;
  name: string;
  token: string;
  appId?: string;
  envId?: string;
  teamId?: string;
  scope: "env" | "team";
}
