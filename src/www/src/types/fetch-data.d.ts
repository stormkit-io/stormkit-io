declare type FetchDataFunc = (
  match: Record<string, string>
) => Promise<{ head: SEO; context: any }>
