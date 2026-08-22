import fs from 'node:fs'
import path from 'node:path'
import { execFileSync } from 'node:child_process'
import { glob } from 'glob'
import routes from './prerender'

const ORIGIN = 'https://www.stormkit.io'

// Routes that should stay out of search results. Legal pages and the 404 carry
// no search value and dilute the sitemap.
const EXCLUDE = new Set(['/404', '/policies/terms', '/policies/privacy'])

// Priority by section, highest first. Anything unlisted falls back to 0.5.
const PRIORITY: [RegExp, string, string][] = [
  [/^\/$/, '1.0', 'weekly'],
  [/^\/vs-/, '0.9', 'monthly'],
  [/^\/(enterprise|about-us|contact)$/, '0.8', 'monthly'],
  [/^\/mcp$/, '0.9', 'monthly'],
  [/^\/docs/, '0.8', 'weekly'],
  [/^\/tutorials/, '0.7', 'monthly'],
  [/^\/blog/, '0.6', 'weekly'],
]

interface Entry {
  loc: string
  lastmod: string
  changefreq: string
  priority: string
}

const ROOT = path.resolve(import.meta.dirname, '../../../')
const TODAY = new Date().toISOString().slice(0, 10)

// Markdown routes drop the numeric ordering prefix from their filename
// (docs/self-hosting/5-runtimes.md -> /docs/self-hosting/runtimes), so the route
// alone cannot be turned back into a path. Walk the same globs prerender.ts uses
// and index the files by the route they produce.
function markdownByRoute(): Map<string, string> {
  const files = [
    ...glob.sync(path.resolve(ROOT, 'docs/**/*.md')),
    ...glob.sync(path.resolve(ROOT, 'content/blog/*.md')),
    ...glob.sync(path.resolve(ROOT, 'content/tutorials/**/*.md')),
  ]

  return new Map(
    files.map((file) => [
      file
        .replace(ROOT, '')
        .replace(/^\/content/, '')
        .replace('.md', '')
        .replace(/\/[\d]+-/, '/'),
      file,
    ])
  )
}

const MARKDOWN = markdownByRoute()

// lastMod prefers the file's last commit date so the sitemap reflects real
// edits. A shallow clone has no history to read, so fall back to mtime and then
// to today rather than failing the build.
function lastMod(route: string): string {
  const file = MARKDOWN.get(route)

  if (!file) {
    return TODAY
  }

  try {
    const out = execFileSync('git', ['log', '-1', '--format=%cs', '--', file], {
      cwd: ROOT,
      encoding: 'utf-8',
      stdio: ['ignore', 'pipe', 'ignore'],
    }).trim()

    if (/^\d{4}-\d{2}-\d{2}$/.test(out)) {
      return out
    }
  } catch {
    // git unavailable or no history for this path
  }

  return fs.statSync(file).mtime.toISOString().slice(0, 10)
}

function entryFor(route: string): Entry {
  const match = PRIORITY.find(([re]) => re.test(route))

  return {
    loc: route === '/' ? `${ORIGIN}/` : `${ORIGIN}${route}`,
    lastmod: lastMod(route),
    changefreq: match ? match[2] : 'monthly',
    priority: match ? match[1] : '0.5',
  }
}

export function generateSitemap(): string {
  const seen = new Set<string>()
  const entries: Entry[] = []

  for (const { route } of routes) {
    if (EXCLUDE.has(route) || seen.has(route)) {
      continue
    }

    seen.add(route)
    entries.push(entryFor(route))
  }

  entries.sort((a, b) => Number(b.priority) - Number(a.priority) || a.loc.localeCompare(b.loc))

  const body = entries
    .map(
      (e) =>
        `  <url>\n` +
        `    <loc>${e.loc}</loc>\n` +
        `    <lastmod>${e.lastmod}</lastmod>\n` +
        `    <changefreq>${e.changefreq}</changefreq>\n` +
        `    <priority>${e.priority}</priority>\n` +
        `  </url>`
    )
    .join('\n')

  return (
    `<?xml version="1.0" encoding="UTF-8"?>\n` +
    `<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">\n` +
    `${body}\n` +
    `</urlset>\n`
  )
}

const dist = path.resolve(import.meta.dirname, '../public/sitemap.xml')
fs.writeFileSync(dist, generateSitemap(), 'utf-8')
console.info(`✅ sitemap.xml written with ${routes.length} candidate routes`)
