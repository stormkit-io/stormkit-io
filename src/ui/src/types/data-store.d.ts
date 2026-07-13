declare type CollectionRecord = {
  id: string;
  value: Record<string, unknown>;
  createdAt: number;
};

declare interface Collection {
  name: string;
  count: number;
}
