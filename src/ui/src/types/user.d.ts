declare type SubscriptionName = "free" | "premium" | "ultimate";
declare type Edition = "community" | "enterprise";

declare interface User {
  id: string;
  avatar: string;
  email: string;
  fullName: string;
  displayName: string;
  memberSince: number;
  isAdmin?: boolean;
  package: {
    id: SubscriptionName;
    name: string;
  };
}

declare interface License {
  enterpise: boolean;
  seats: number;
  raw: string;
}

declare interface ConnectedAccount {
  provider: Provider;
  hasPersonalAccessToken: boolean;
  displayName: string;
  url?: string;
}

declare interface InstanceDetails {
  latest?: {
    apiVersion: string;
  };
  license?: {
    seats: number;
    remaining: number;
    edition: Edition;
  };
  stormkit?: {
    apiCommit: string;
    apiVersion: string;
    edition: "development" | "self-hosted" | "cloud";
  };
  update?: {
    api: boolean;
  };
  auth?: {
    github?: string; // The github account name
  };
}

declare interface UserMetrics {
  max: {
    bandwidthInBytes: number;
    buildMinutes: number;
    functionInvocations: number;
    storageInBytes: number;
  };
  used: {
    bandwidthInBytes: number;
    buildMinutes: number;
    functionInvocations: number;
    storageInBytes: number;
  };
}
