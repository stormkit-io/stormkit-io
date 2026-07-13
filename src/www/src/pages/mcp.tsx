import DocTitle from '~/pages/docs/[category]/[title]'
import { fetchData as docsFetchData } from '~/pages/docs/[category]/_ssr'

// Top-level /mcp alias for the API/MCP documentation page. The www server
// rewrites /mcp -> /docs/api/mcp, but the URL stays /mcp, so the client router
// needs a matching route that renders the same doc instead of falling through
// to the 404 catch-all.

// Required for SSR — always load the api/mcp doc regardless of route params.
export const fetchData: FetchDataFunc = async () =>
  docsFetchData({ category: 'api', title: 'mcp' })

export default function MCP() {
  return <DocTitle category="api" title="mcp" />
}
