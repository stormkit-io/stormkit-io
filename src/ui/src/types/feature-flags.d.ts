declare interface FeatureFlag {
  flagName: string;
  flagValue: boolean;
}

interface SK {
  features: Record<string, boolean>;
}

declare var sk: SK | undefined;
