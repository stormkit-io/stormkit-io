import { StaticRouter } from 'react-router-dom'
import { renderToString } from 'react-dom/server'
import createEmotionServer from '@emotion/server/create-instance'
import fs from 'node:fs'
import path from 'node:path'
import http from 'node:http'
import { fileURLToPath } from 'node:url'
import serverless from '@stormkit/serverless'
import createRoutes from './routes'
import Context from './context'
import App from './App'
import createEmotionCache from './emotion-cache'

interface RenderReturn {
  status: number
  content: string
  head: string
}

export type RenderFunction = (url: string, seo?: SEO) => Promise<RenderReturn>

const defaultSEO: SEO = {
  title: 'Self-Hosted Vercel and Netlify Alternative',
  description:
    'Stormkit is a self-hostable, full-featured alternative to Vercel and Netlify, offering powerful features such as deployment previews, multiple environments, and seamless integration with Git.',
  type: 'website',
  domain: {
    name: 'Stormkit.io',
    url: 'https://www.stormkit.io',
  },
  twitter: {
    card: 'summary_large_image',
    creator: '@stormkitio',
    imageContentType: 'image/png',
    imageLarge: 'https://www.stormkit.io/stormkit-og-image-1200x628.png',
    imageTitle: 'Stormkit OG Image',
    imageHeight: 628,
    imageWidth: 1200,
  },
}

export const render: RenderFunction = async (url, seo) => {
  const cache = createEmotionCache()
  const { extractCriticalToChunks, constructStyleTagsFromChunks } =
    createEmotionServer(cache)

  const { routes, head, context } = await createRoutes(url)

  const tags = {
    ...defaultSEO,
    ...JSON.parse(JSON.stringify(head || {})),
    ...JSON.parse(JSON.stringify(seo || {})),
  }

  // Prefix the title with the domain.name property.
  tags.title = tags.title + ` - Stormkit`

  const content =
    renderToString(
      <Context.Provider value={{ ...context, url, isHomePage: url === '/' }}>
        <StaticRouter location={{ pathname: url }}>
          <App routes={routes} cache={cache} />
        </StaticRouter>
      </Context.Provider>
    ) +
    (context
      ? `<script>window.CONTEXT = ${JSON.stringify(context)}</script>`
      : '')

  // Grab the CSS from emotion
  const emotionChunks = extractCriticalToChunks(content)
  const emotionCss = constructStyleTagsFromChunks(emotionChunks)

  const title = tags.title.replace(/^["']|["']$/g, '')

  return {
    status: 200,
    content,
    head: [
      `<title>${title}</title>`,
      `<meta charset="utf-8" />`,
      `<meta name="viewport" content="width=device-width, initial-scale=1" />`,
      `<meta name="description" content="${tags.description}" />`,
      `<meta property="og:title" content="${title}" />`,
      `<meta property="og:type" content="${tags.type}" />`,
      `<meta property="og:url" content="${tags.domain?.url}" />`,
      `<meta property="og:description" content="${tags.description}" />`,
      `<meta property="og:image" content="${tags.domain?.url}/stormkit-og-image.png" />`,
      `<meta name="twitter:card" content="${tags.twitter!.card}" />`,
      `<meta name="twitter:creator" content="${tags.twitter!.creator}" />`,
      `<meta name="twitter:title" content="${title}" />`,
      `<meta name="twitter:description" content="${tags.description}" />`,
      `<meta name="twitter:image:content_type" content="${tags.twitter.imageContentType}"/>`,
      `<meta name="twitter:image:title" content="${
        tags.twitter!.imageTitle
      }" />`,
      `<meta name="twitter:image" content="${tags.twitter!.imageLarge}" />`,
      `<meta name="twitter:image:width" content="${
        tags.twitter!.imageWidth
      }"/>`,
      `<meta name="twitter:image:height" content="${
        tags.twitter!.imageWidth
      }"/>`,
      `<link rel="apple-touch-icon" href="/stormkit-logo.png"/>`,
      `<link rel="icon" type="image/svg+xml" href="/stormkit-logo.png" />`,
      `<link href="/src/index.css" rel="stylesheet" />`,
      emotionCss,
    ]
      .join('\n')
      .trim(),
  }
}

// This handler add support for Stormkit environment. This is
// the entry point of the serverless application.
export const handler = serverless(
  async (req: http.IncomingMessage, res: http.ServerResponse) => {
    // We are in assets folder
    const dir = path.dirname(fileURLToPath(import.meta.url))
    const html = fs.readFileSync(path.join(dir, './index.html'), 'utf-8')

    const { status, content, head } = await render(
      req.url?.split(/\?#/)[0] || '/'
    )

    res.writeHead(status, 'OK', { 'Content-Type': 'text/html; charset=utf-8' })
    res.write(
      html
        .replace('</head>', `${head}</head>`)
        .replace(`<div id="root"></div>`, `<div id="root">${content}</div>`)
    )
    res.end()
  }
)
