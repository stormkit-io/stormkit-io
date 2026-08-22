import fs from 'node:fs'
import path from 'node:path'

/**
 * The OpenAPI document lives next to the API it describes
 * (src/ce/api/public/v1/openapi.json), where a Go test keeps it in sync with the
 * registered routes. The marketing site publishes the very same bytes at
 * https://www.stormkit.io/openapi.json, which is where agents look first.
 */

const SOURCE = path.resolve(
  import.meta.dirname,
  '../../ce/api/public/v1/openapi.json'
)

const TARGET = path.resolve(import.meta.dirname, '../public/openapi.json')

export function copyOpenAPI(): string {
  const spec = fs.readFileSync(SOURCE, 'utf-8')

  // Parsing before writing turns a malformed spec into a failed build rather
  // than a broken file served to agents.
  JSON.parse(spec)

  fs.writeFileSync(TARGET, spec, 'utf-8')

  return TARGET
}

if (process.argv[1] && process.argv[1].endsWith('copy-openapi.ts')) {
  copyOpenAPI()
  console.info('✅ openapi.json copied to public/')
}
