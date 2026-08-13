export const DEFAULT_ORIGIN = 'https://www.stormkit.io'

/**
 * canonicalUrl builds the absolute, canonical form of a page: the www origin, no
 * query string or fragment, and no trailing slash except on the root. Without
 * this, search engines treat /page, /page/, and /page?ref=x as three separate
 * URLs and split the ranking signals between them.
 */
export function canonicalUrl(pathname: string, origin?: string): string {
  const base = (origin || DEFAULT_ORIGIN).replace(/\/+$/, '')
  const clean = (pathname || '/').split(/[?#]/)[0].replace(/\/+$/, '')

  return clean === '' ? `${base}/` : `${base}${clean}`
}
