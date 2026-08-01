export interface FilterOption {
  value: string;
  text: string;
}

export type FilterKind = "text" | "enum" | "number" | "datetime";

export interface FilterDef {
  /** Query parameter name. Doubles as the token identity, so it must be unique. */
  key: string;
  label: string;
  kind: FilterKind;
  /** Static suggestions, used by `enum` filters. */
  options?: FilterOption[];
  /** Remote suggestions. Suggestions are advisory — typed values are always accepted. */
  search?: (term: string) => Promise<FilterOption[]>;
  /** Shown above the suggestion list to qualify where suggestions come from. */
  searchHint?: string;
  /** Renders the committed value on the token. */
  format?: (value: string) => string;
  /** Applied before a typed value is committed. */
  normalize?: (value: string) => string;
}

/** Committed filters, keyed by `FilterDef.key`. One value per filter. */
export type FilterValues = Record<string, string>;
