import fs from 'node:fs'
import path from 'node:path'
import { glob } from 'glob'
import { parseAttributes } from '../src/helpers/markdown'

/**
 * Every page that is written in markdown gets published as markdown as well, at
 * the same URL plus a `.md` suffix. The hosting layer then serves that file to
 * clients that send `Accept: text/markdown` (acceptmarkdown.com), so an agent
 * reads the source instead of scraping the rendered page.
 *
 * The mapping from file to route is the same one prerender.ts uses; both walk
 * the same globs so a page can never be rendered without its markdown twin.
 */

const ORIGIN = 'https://www.stormkit.io'
const ROOT = path.resolve(import.meta.dirname, '../../../')
const PUBLIC_DIR = path.resolve(import.meta.dirname, '../public')

interface MarkdownPage {
  route: string
  file: string
}

function toRoute(filePath: string): string {
  return filePath
    .replace(ROOT, '')
    .replace(/^\/content/, '')
    .replace('.md', '')
    .replace(/\/[\d]+-/, '/')
}

function pages(): MarkdownPage[] {
  return [
    ...glob.sync(path.resolve(ROOT, 'docs/**/*.md')),
    ...glob.sync(path.resolve(ROOT, 'content/blog/*.md')),
    ...glob.sync(path.resolve(ROOT, 'content/tutorials/**/*.md')),
  ].map((file) => ({ route: toRoute(file), file }))
}

/**
 * Front matter is YAML meant for the renderer, not for a reader. It is replaced
 * by a plain markdown header carrying the same information plus the canonical
 * URL, so the file makes sense on its own once an agent has fetched it.
 */
export function toPublishedMarkdown(content: string, route: string): string {
  const attributes = parseAttributes(content)
  const body = content.replace(/^---\r?\n[\s\S]*?\r?\n---\r?\n?/, '').trim()
  const header = [`<!-- Source: ${ORIGIN}${route} -->`]

  if (attributes.title) {
    header.push(`<!-- Title: ${attributes.title} -->`)
  }

  if (attributes.description) {
    header.push(`<!-- Description: ${attributes.description} -->`)
  }

  return `${header.join('\n')}\n\n${body}\n`
}

export function generateMarkdown(): number {
  let written = 0

  for (const page of pages()) {
    const target = path.join(PUBLIC_DIR, `${page.route}.md`)
    const content = toPublishedMarkdown(
      fs.readFileSync(page.file, 'utf-8'),
      page.route
    )

    fs.mkdirSync(path.dirname(target), { recursive: true })
    fs.writeFileSync(target, content, 'utf-8')
    written++
  }

  return written
}

// Only run when executed directly, so the tests can import the helpers.
if (process.argv[1] && process.argv[1].endsWith('generate-markdown.ts')) {
  const written = generateMarkdown()
  console.info(`✅ ${written} markdown pages written to public/`)
}
