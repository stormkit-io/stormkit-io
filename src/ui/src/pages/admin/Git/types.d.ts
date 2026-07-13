export interface GitDetails {
  github?: {
    appId: string;
    account: string;
    clientId: string;
    hasClientSecret: boolean;
    hasPrivateKey: boolean;
  };
  gitlab?: {
    clientId: string;
    hasClientSecret: boolean;
    redirectUrl: string;
  };
  bitbucket?: {
    clientId: string;
    hasDeployKey?: boolean;
    hasClientSecret: boolean;
  };
}
