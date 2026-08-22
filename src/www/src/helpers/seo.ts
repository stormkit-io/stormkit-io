export const DEFAULT_ORIGIN = 'https://www.stormkit.io'

/**
 * canonicalUrl builds the absolute, canonical form of a page: the www origin, no
 * query string or fragment, and no trailing slash except on the root. Without
 * this, search engines treat /page, /page/, and /page?ref=x as three separate
 * URLs and split the ranking signals between them.
 */
/**
 * markdownAlternate returns the URL of the markdown representation of a page, or
 * an empty string when the page has none. Only pages that are written in
 * markdown have one — scripts/generate-markdown.ts publishes exactly those — so
 * the link never points at a URL the build did not produce.
 */
export function markdownAlternate(pathname: string): string {
  const clean = (pathname || '/').split(/[?#]/)[0].replace(/\/+$/, '')

  if (clean === '') {
    return '/index.md'
  }

  return /^\/(docs|blog|tutorials)\/.+/.test(clean) ? `${clean}.md` : ''
}

export function canonicalUrl(pathname: string, origin?: string): string {
  const base = (origin || DEFAULT_ORIGIN).replace(/\/+$/, '')
  const clean = (pathname || '/').split(/[?#]/)[0].replace(/\/+$/, '')

  return clean === '' ? `${base}/` : `${base}${clean}`
}
