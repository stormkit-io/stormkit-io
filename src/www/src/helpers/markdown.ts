import type { NavigationItem } from '~/components/DocsNav/DocsNav'
import { useMemo, useEffect, useState, useContext } from 'react'
import markdown from 'markdown-it'
import prism from 'markdown-it-prism'
import Context from '~/context'
import 'prismjs/components/prism-go'
import 'prismjs/components/prism-diff'
import 'prismjs/components/prism-json'
import 'prismjs/components/prism-bash'
import 'prismjs/components/prism-yaml'
import 'prismjs/components/prism-regex'
import 'prismjs/components/prism-typescript'
import 'prismjs/components/prism-docker'
import { useParams } from 'react-router'

export interface Attributes {
  title?: string
  subtitle?: string
  date?: string // yyyy-mm-dd
  description?: string
  keywords?: string
  category?: string
  draft?: string
  authorName?: string
  authorTw?: string // Twitter handle
  authorImg?: string // URL to profile picture
  search?: string // Whether to include as a resource in the search modal or not
}

export const parseAttributes = (
  content: string,
  category?: string
): Attributes => {
  const attrs: Attributes = {}
  const index = content.indexOf('---', 2)

  if (index > -1) {
    content
      .slice(3, index)
      .split(/\n/g)
      .filter((i) => i)
      .forEach((str) => {
        const [key, ...value] = str.split(':')
        attrs[
          // author-name => authorName
          key
            .toLowerCase()
            .replace(/-[a-z]/, (m) =>
              m.toUpperCase().replace('-', '')
            ) as keyof Attributes
        ] = value.join(':').trim()
      })
  }

  if (category) {
    attrs.category = category
  }

  return attrs
}

export function toTitleCase(str: string) {
  const ignore = ['and', 'by', 'then', 'or']

  return str.replace(/\w\S*/g, function (txt) {
    if (ignore.indexOf(txt.toLowerCase()) > -1) {
      return txt
    }

    return txt.charAt(0).toUpperCase() + txt.substring(1).toLowerCase()
  })
}

const md = new markdown({ html: true, linkify: true }).use(prism)

interface WithContentOptions {
  defaultTitle?: string
  defaultCategory?: string
}

export function withContent(
  fetchData: FetchDataFunc,
  options?: WithContentOptions
) {
  const [isClientSide, setIsClientSide] = useState(false)
  const context = useContext(Context) || {}
  const params = useParams()
  const [nav, setNav] = useState<NavigationItem[]>(context.navigation || [])
  const [content, setContent] = useState<string>(context.content || '')
  const [loading, setLoading] = useState(isClientSide)

  useEffect(() => {
    setLoading(true)
    setIsClientSide(true)

    fetchData({
      slug: params.slug || '',
      title: params.title || options?.defaultTitle || '',
      category: params.category || options?.defaultCategory || '',
    })
      .then((res) => {
        setNav(res.context.navigation)
        setContent(res.context.content)
      })
      .finally(() => {
        setLoading(false)
      })
  }, [params])

  const markdownContent = useMemo(() => {
    let __html: string

    // We get the content directly from the server side rendered content because
    // md.render does not behave the same server-side and client-side.
    if (isClientSide === false && typeof document !== 'undefined') {
      __html = document.querySelector('#blog-content')?.innerHTML || ''
    } else {
      __html = md.render(content || '')
    }

    return {
      __html,
    }
  }, [content, isClientSide])

  return { content: markdownContent, navigation: nav, loading }
}
