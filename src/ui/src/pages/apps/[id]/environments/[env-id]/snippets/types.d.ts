declare interface Snippet {
  title: string;
  content: string;
  enabled: boolean;
  prepend: boolean;
  location?: "head" | "body";
  rules?: SnippetRules;
  id?: string;
}

declare interface SnippetRules {
  hosts?: string[];
  path?: string;
}
