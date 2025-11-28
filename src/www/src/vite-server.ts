import type { RenderFunction } from './entry-server'
import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import express from 'express'
import dotenv from 'dotenv'
import { createServer as createViteServer } from 'vite'
import apiHandler from '@stormkit/serverless/middlewares/express'

dotenv.config()

const __dirname = path.dirname(fileURLToPath(import.meta.url))

function injectContent(head: string, content: string, template: string) {
  return template
    .replace(`</head>`, `${head}</head>`)
    .replace(`<div id="root"></div>`, `<div id="root">${content}</div>`)
}

async function createServer() {
  const app = express()

  // Create Vite server in middleware mode and configure the app type as
  // 'custom', disabling Vite's own HTML serving logic so parent server
  // can take control
  const vite = await createViteServer({
    server: { middlewareMode: true },
    appType: 'custom',
  })

  // Generate docs
  const { default: docs } = await vite.ssrLoadModule('./src/generate-docs')

  docs()

  // use vite's connect instance as middleware
  // if you use your own express router (express.Router()), you should use router.use
  app.use(vite.middlewares)

  // Add support for a local environment API.
  app.all(
    ['/api', '/api/*'],
    apiHandler({
      apiDir: path.join(__dirname, 'api'),
      moduleLoader: vite.ssrLoadModule,
    })
  )

  app.get('*', async (req, res, next) => {
    try {
      const url: string = req.originalUrl.split(/\?#/)[0] || '/'

      // // 1. Read and apply Vite HTML transforms. This injects the Vite HMR client, and
      // //    also applies HTML transforms from Vite plugins, e.g. global preambles
      // //    from @vitejs/plugin-react
      const template: string = await vite.transformIndexHtml(
        req.originalUrl,
        fs.readFileSync(path.resolve(__dirname, '..', 'index.html'), 'utf-8')
      )

      // 2. Load the server entry. vite.ssrLoadModule automatically transforms
      //    your ESM source code to be usable in Node.js! There is no bundling
      //    required, and provides efficient invalidation similar to HMR.
      const { render } = (await vite.ssrLoadModule('./src/entry-server')) as {
        render: RenderFunction
      }

      const rendered = await render(url)

      return res
        .status(rendered.status)
        .set({ 'Content-Type': 'text/html' })
        .send(injectContent(rendered.head, rendered.content, template))
    } catch (e) {
      // If an error is caught, let Vite fix the stack trace so it maps back to
      // your actual source code.
      if (e instanceof Error) {
        console.log(e)
        vite.ssrFixStacktrace(e)
      }

      next(e)
    }
  })

  app.listen(5173, () => {
    console.log(`Server listening on http://localhost:5173`)
  })
}

async function generateDocs() {
  // Create Vite server to load the routes files
  const vite = await createViteServer({
    server: { middlewareMode: true },
    appType: 'custom',
  })

  // Generate docs
  const { default: docs } = await vite.ssrLoadModule('./src/generate-docs')

  docs()

  await vite.close()
}

interface Prerender {
  route: string
  title?: string
  description?: string
}

async function generateStaticPages() {
  // Create Vite server to load the routes files
  const vite = await createViteServer({
    server: { middlewareMode: true },
    appType: 'custom',
  })

  const { default: routesToPrerender } = (await vite.ssrLoadModule(
    './src/prerender'
  )) as { default: Prerender[] }

  let render: RenderFunction

  try {
    const imported = await vite.ssrLoadModule('./src/entry-server')
    render = imported.render as RenderFunction
  } catch (e) {
    throw new Error(
      "Failed to load the render function from 'entry-server'. " + e
    )
  }

  const dist = path.join(path.dirname(__dirname), '.stormkit/public')

  if (!fs.existsSync(dist)) {
    throw new Error(
      '.stormkit/public is not available. Did you run `npm run build:spa`?'
    )
  }

  const template = fs.readFileSync(path.join(dist, 'index.html'), 'utf-8')
  const manifest = JSON.parse(
    fs.readFileSync(path.join(dist, '.vite', 'manifest.json'), 'utf-8')
  )

  // Push `/` to the end if any.
  routesToPrerender.sort((a, b) => {
    return a.route === '/' || b.route === '/' ? -1 : 0
  })

  for (const r of routesToPrerender) {
    const data = await render(r.route, {
      title: r.title,
      description: r.description,
    })

    const fileName = r.route.endsWith('/')
      ? `${r.route}index.html`
      : `${r.route}.html`

    const absPath = path.join(dist, fileName)
    let content = injectContent(data.head, data.content, template)

    // Fix the path to the static assets.
    Object.keys(manifest).forEach((fileName) => {
      if (fileName.startsWith('src/assets')) {
        content = content.replaceAll(fileName, manifest[fileName].file)
      }
    })

    // Fix css files
    content = content.replace('src/index.css', manifest['index.html'].css[0])

    fs.mkdirSync(path.dirname(absPath), { recursive: true })
    fs.writeFileSync(absPath, content, 'utf-8')

    console.log(`Prerendered: ${fileName}`)
  }

  // fs.unlinkSync(path.join(dist, '.vite', 'manifest.json'))

  await vite.close()
}

;(async () => {
  if (process.env.SSG === 'true') {
    console.info('🟢 Generating static routes...')
    await generateStaticPages()
    console.info('✅ Static routes generated')
  } else if (process.env.DOCS === 'true') {
    console.info('🟢 Generating docs...')
    await generateDocs()
    console.info('✅ Docs generated')
  } else {
    createServer()
  }
})()
