import { useEffect, useState } from 'react'
import Box from '@mui/material/Box'
import Typography from '@mui/material/Typography'
import Link from '@mui/material/Link'
import Header from '~/components/Header'
import Footer from '~/components/Footer'
import DocSearch from '~/components/DocsNav/DocSearch'

// A dead URL is usually a typo or a stale link, so search is the fastest way
// out and takes the primary position. The links under it are the destinations
// someone would otherwise go hunting for in the nav. Machine-readable
// resources are deliberately absent: agents are served /404.md, which lists
// them, and putting them here only dilutes the three a reader wants.
const LINKS: { href: string; text: string }[] = [
  { href: '/docs/welcome/getting-started', text: 'Documentation' },
  { href: '/docs/api/authentication', text: 'API documentation' },
  { href: '/', text: 'Homepage' },
]

/**
 * RequestedPath shows the path that produced the 404, which is what lets a
 * reader spot their own typo. It renders only after mount: this page is
 * prerendered to /404.html and served for every missing URL, so at build time
 * the path is not knowable and rendering one would state a falsehood.
 */
function RequestedPath() {
  const [path, setPath] = useState('')

  useEffect(() => {
    setPath(window.location.pathname)
  }, [])

  if (!path) {
    return null
  }

  return (
    <Typography
      component="p"
      sx={{
        mt: 1.5,
        fontSize: 14,
        color: 'text.secondary',
        wordBreak: 'break-all',
      }}
    >
      Nothing is served at{' '}
      <Box
        component="code"
        sx={{
          px: 0.75,
          py: 0.25,
          borderRadius: 1,
          bgcolor: 'page.transparent',
          fontFamily: 'monospace',
          color: 'primary.contrastText',
        }}
      >
        {path}
      </Box>
    </Typography>
  )
}

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
          py: { xs: 6, md: 10 },
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
        }}
        maxWidth="lg"
      >
        <Box sx={{ textAlign: 'center', maxWidth: 560 }}>
          <Typography
            variant="h1"
            sx={{
              fontWeight: 'bold',
              fontSize: { xs: 56, md: 80 },
              lineHeight: 1.1,
              color: 'secondary.dark',
            }}
          >
            4 oh 4
          </Typography>
          <Typography
            variant="h4"
            component="p"
            sx={{ mt: 2, fontSize: { xs: 20, md: 28 } }}
          >
            There is nothing under this link
          </Typography>
          <RequestedPath />

          <Box
            sx={{
              mt: 5,
              display: 'flex',
              justifyContent: 'center',
              // DocSearch lays itself out for the docs nav, where it sits at the
              // start of a flex row. Centre it here without touching that.
              '& > div': { flex: 'none', justifyContent: 'center' },
              '& .MuiTextField-root': { minWidth: { xs: '100%', md: 360 } },
            }}
          >
            <DocSearch />
          </Box>

          <Box
            component="ul"
            sx={{
              m: 0,
              mt: 4,
              p: 0,
              listStyle: 'none',
              display: 'flex',
              flexWrap: 'wrap',
              justifyContent: 'center',
              columnGap: 3,
              rowGap: 1,
            }}
          >
            {LINKS.map((link) => (
              <Box component="li" key={link.href}>
                <Link href={link.href} underline="hover" sx={{ fontSize: 15 }}>
                  {link.text}
                </Link>
              </Box>
            ))}
          </Box>
        </Box>
      </Box>
      <Footer />
    </Box>
  )
}
