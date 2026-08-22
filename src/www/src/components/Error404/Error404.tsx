import Box from '@mui/material/Box'
import Typography from '@mui/material/Typography'
import Link from '@mui/material/Link'
import Header from '~/components/Header'
import Footer from '~/components/Footer'

// A dead URL is where both people and agents most need a way out, so the page
// names the entry points explicitly instead of only apologising. The same list
// is served as markdown from /404.md.
const LINKS: { href: string; text: string; hint: string }[] = [
  {
    href: '/docs/welcome/getting-started',
    text: 'Documentation',
    hint: 'Guides and reference',
  },
  {
    href: '/docs/api/authentication',
    text: 'API documentation',
    hint: 'Authenticate and call the REST API',
  },
  { href: '/mcp', text: 'MCP server', hint: 'Drive Stormkit from an agent' },
  { href: '/sitemap.xml', text: 'Sitemap', hint: 'Every published page' },
  { href: '/llms.txt', text: 'llms.txt', hint: 'What Stormkit is, in one file' },
]

export default function Error404() {
  return (
    <Box
      sx={{
        minHeight: '100vh',
        display: 'flex',
        flexDirection: 'column',
        bgcolor: 'background.default',
        color: 'primary.contrastText',
      }}
    >
      <Header />
      <Box
        sx={{
          flex: 1,
          m: 'auto',
          px: { xs: 2, md: 0 },
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
        }}
        maxWidth="lg"
      >
        <Box sx={{ textAlign: 'center' }}>
          <Typography
            variant="h1"
            color="secondary"
            sx={{ fontWeight: 'bold' }}
          >
            4 oh 4
          </Typography>
          <Typography variant="h4" component="p" sx={{ mt: 4 }}>
            There is nothing under this link
          </Typography>
          <Typography
            variant="h2"
            sx={{ mt: 6, mb: 2, fontSize: 16, fontWeight: 600 }}
          >
            Where to look next
          </Typography>
          <Box
            component="ul"
            sx={{
              m: 0,
              p: 0,
              listStyle: 'none',
              display: 'flex',
              flexDirection: 'column',
              gap: 1,
            }}
          >
            {LINKS.map((link) => (
              <Box component="li" key={link.href}>
                <Link href={link.href} color="secondary" underline="hover">
                  {link.text}
                </Link>
                <Typography
                  component="span"
                  sx={{ ml: 1, opacity: 0.6, fontSize: 14 }}
                >
                  {link.hint}
                </Typography>
              </Box>
            ))}
          </Box>
        </Box>
      </Box>
      <Footer />
    </Box>
  )
}
