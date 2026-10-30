declare interface DiffFields {
  appName?: string;
  appRepo?: string;
  envId?: string;
  envName?: string;
  envBranch?: string;
  envBuildConfig?: BuildConfig;
  envAutoPublish?: boolean;
  envAutoDeploy?: boolean;
  envAutoDeployBranches?: string;
  domainName?: string;
  domainCertValue?: string;
  domainCertKey?: string;
  snippetTitle?: string;
  snippetContent?: string;
  snippetEnabled?: boolean;
  snippetPrepend?: boolean;
  snippetRules?: SnippetRules;
  snippetLocation?: "head" | "body";
  snippets?: string[]; // Name of created snippets.
  authWallStatus?: string;
  authWallCreateLoginEmail?: string;
  authWallCreateLoginID?: string;
  authWallDeleteLoginIDs?: string;
  authWallLoginEmail?: string;
  authWallLoginApps?: string;
  authWallLoginMetadata?: string;
  authWallPasswordChanged?: boolean;
  authWallLoginMagicLink?: boolean;
  deploymentId?: string;
  mailerHost?: string;
  mailerPort?: string;
  mailerUsername?: string;
  mailerPasswordChanged?: boolean;
  mailerUseDefault?: boolean;
  authProviderName?: string;
  authProviderStatus?: boolean;
  authProviderFromAddress?: string;
  authProviderClientId?: string;
  autoPublished?: boolean;
  restarted?: boolean;
  stopped?: boolean;
  apiKeyName?: string;
  apiKeyScope?: string;
  apiKeyReadOnly?: boolean;
}

declare interface Diff {
  old: DiffFields;
  new: DiffFields;
}

declare interface Audit {
  id: string;
  appId?: string;
  envId?: string;
  envName?: string;
  teamId?: string;
  userDisplay?: string;
  tokenName?: string;
  action: string;
  diff: Diff;
  timestamp?: int;
}
