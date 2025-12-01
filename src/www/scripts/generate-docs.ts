import * as fs from 'node:fs'
import * as path from 'node:path'
import { glob } from 'glob'
import { parseAttributes, toTitleCase } from '../src/helpers/markdown'

// @ts-ignore
const dirname = import.meta.dirname
const files = glob.sync(path.resolve(dirname, '../../../docs/**/*.md'))

interface Doc {
  id: number
  pageTitle?: string // This is the title used in the
  contentTitle?: string
  description?: string
  url?: string
  keywords?: string
}

export default async function generateDocs() {
  const documents: Doc[] = []

  files.map((file, index) => {
    const content = fs.readFileSync(file, 'utf-8')
    const metadata = parseAttributes(content)

    if (!metadata) {
      return console.error(`Error parsing file: ${file}`)
    }

    const category = path.basename(path.dirname(file))
    const fileNameWithoutExtension = path.basename(file, path.extname(file))
    const fileNameWithoutSort = fileNameWithoutExtension.replace(/^\d+-/, '')

    const url = `/docs/${category}/${fileNameWithoutSort}`

    if (process.env.DOCS === 'true') {
      console.log(`Generating doc for: ${url}`)
    }

    documents.push({
      id: index + 1,
      pageTitle: toTitleCase(fileNameWithoutSort.replace(/-/g, ' ')!),
      contentTitle: metadata.title,
      description: metadata.description,
      keywords: metadata.keywords,
      url,
    })

    return true
  })

  const output = JSON.stringify(documents, null, 2)
  fs.writeFileSync('src/search-docs.json', output, 'utf8')
}
