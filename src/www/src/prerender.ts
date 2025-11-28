interface Prerender {
  route: string
  title?: string
  description?: string
}

const docs = import.meta.glob('/content/docs/**/*.md', {
  query: '?raw',
  import: 'default',
})
const blog = import.meta.glob('/content/blog/*.md', {
  query: '?raw',
  import: 'default',
})

const routes: Prerender[] = [
  { route: '/' },
  { route: '/404' },
  { route: '/contact' },
  { route: '/enterprise' },
  { route: '/about-us' },
  { route: '/vs-vercel' },
  { route: '/vs-netlify' },
  { route: '/tutorials/how-to-deploy-your-self-hosted-strapi-instance' },
  { route: '/tutorials/how-to-self-host-stormkit-on-hetzner-cloud' },
  { route: '/tutorials/how-to-deploy-your-self-hosted-remix-app' },
  { route: '/tutorials/setting-up-stormkit-with-cloudflare-tunnels' },
  {
    route:
      '/tutorials/how-to-run-automated-e2-e-tests-with-stormkit-and-browserless',
  },
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

  // Prerender docs
  { route: '/docs/welcome/getting-started' },
  ...Object.keys(docs).map((file) => ({
    route: file
      .replace('/content', '')
      .split('--')
      .join('/')
      .replace(/\/\d+-/, '/')
      .replace('.md', ''),
  })),

  // Prerender blog
  { route: '/blog' },
  ...Object.keys(blog).map((file) => ({
    route: file
      .replace('/content', '')
      .replace('.md', '')
      .replace(/--[\d]+/, ''),
  })),
]

export default routes
