import path from 'node:path'
import { glob } from 'glob'

interface Prerender {
  route: string
  title?: string
  description?: string
}

// @ts-ignore
const dirname = import.meta.dirname
const root = path.resolve(dirname, '../../../')
const docs = glob.sync(path.resolve(root, 'docs/**/*.md'))
const blog = glob.sync(path.resolve(root, 'content/blog/*.md'))
const tuts = glob.sync(path.resolve(root, 'content/tutorials/**/*.md'))

function toSlug(filePath: string) {
  return {
    route: filePath
      .replace(root, '')
      .replace(/^\/content/, '')
      .replace('.md', '')
      .replace(/\/[\d]+/, ''),
  }
}

const routes: Prerender[] = [
  { route: '/' },
  { route: '/404' },
  { route: '/contact' },
  { route: '/enterprise' },
  { route: '/about-us' },
  { route: '/vs-vercel' },
  { route: '/vs-netlify' },
  {
    route: '/policies/terms',
    title: 'Terms of Service',
    description: 'Read terms of service before using Stormkit',
  },
  {
    route: '/policies/privacy',
    title: 'Privacy policy',
    description: 'Read our privacy policy',
  },

  ...docs.map(toSlug),
  ...blog.map(toSlug),
  ...tuts.map(toSlug),
]

export default routes
