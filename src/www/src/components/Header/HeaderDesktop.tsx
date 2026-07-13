import { useContext } from 'react'
import Box from '@mui/material/Box'
import Button from '@mui/material/Button'
import GitHubIcon from '@mui/icons-material/GitHub'
import ArrowForwardIcon from '@mui/icons-material/ArrowForward'
import Context from '~/context'
import MenuIconDesktop from './MenuIconDesktop'
import LinksDesktop from './LinksDesktop'

export default function Header() {
  const { isHomePage } = useContext(Context)
  const headerBgColor = '#17172b'

  return (
    <Box
      maxWidth={isHomePage ? 'xl' : 'auto'}
      sx={{
        mt: isHomePage ? 4 : 0,
        mx: 'auto',
        borderRadius: isHomePage ? 10 : 0,
        display: {
          xs: 'none',
          lg: 'block',
        },
        bgcolor: isHomePage ? headerBgColor : 'page.container',
        boxShadow: 2,
        border: '1px solid rgb(20, 6, 48)',
        px: { xs: 1, md: 0 },
      }}
    >
      <Box
        sx={{
          m: 'auto',
          py: 2,
          px: { xs: 1, md: 4 },
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          maxWidth: '1560px',
          margin: '0 auto',
        }}
      >
        <MenuIconDesktop />
        <Box
          sx={{
            flex: 1,
            fontSize: 12.5,
            display: 'flex',
            alignItems: 'center',
            bgcolor: 'transparent',
            zIndex: 1,
          }}
        >
          <LinksDesktop />
          <Box
            sx={{
              display: 'flex',
              flex: 1,
              justifyContent: 'flex-end',
            }}
          >
            <Button
              variant="outlined"
              color="info"
              href="https://github.com/stormkit-io/stormkit-io"
              rel="noreferrer noopener"
              target="_blank"
              data-umami-event="GitHub"
              sx={{
                display: { xs: 'none', lg: 'inline-flex' },
                mr: 2,
              }}
            >
              <GitHubIcon sx={{ mr: 1 }} />
              <Box
                component="span"
                sx={{ display: { xs: 'none', xl: 'inline' } }}
              >
                GitHub
              </Box>
              <ArrowForwardIcon
                sx={{
                  ml: { xs: 0, xl: 1 },
                  transform: 'rotate(-45deg)',
                  fontSize: 16,
                  display: { xs: 'none', xl: 'inline-block' },
                }}
              />
            </Button>
            <Button
              variant="outlined"
              color="info"
              href="/contact"
              sx={{ mr: 2 }}
            >
              Contact us
            </Button>
            <Button
              variant="outlined"
              color="info"
              href="https://app.stormkit.io"
              data-umami-event="Cloud login"
            >
              Cloud Login
              <ArrowForwardIcon
                sx={{ ml: 1, fontSize: 16, transform: 'rotate(-45deg)' }}
              />
            </Button>
          </Box>
        </Box>
      </Box>
    </Box>
  )
}
