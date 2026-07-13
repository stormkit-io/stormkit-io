import { useState } from 'react'
import Box from '@mui/material/Box'
import Link from '@mui/material/Link'
import IconButton from '@mui/material/IconButton'
import MenuIcon from '@mui/icons-material/Menu'
import StormkitLogoText from '~/assets/logos/stormkit-logo-text-h--white.svg'
import MenuIconMobile from './HeaderMenuMobile'
import LinksMobile from './LinksMobile'

export default function Header() {
  const [isMenuOpen, setIsMenuOpen] = useState<boolean>(false)

  return (
    <Box
      sx={{
        display: { xs: 'block', lg: 'none' },
        bgcolor: 'page.container',
        boxShadow: 2,
        px: 1,
      }}
    >
      <Box
        sx={{
          m: 'auto',
          py: 2,
          px: 1,
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
        }}
      >
        <Link href="/" sx={{ display: 'block', width: 130 }}>
          <Box
            component="img"
            src={StormkitLogoText}
            alt="Stormkit Logo"
            width="100%"
          />
        </Link>
        <IconButton
          onClick={() => setIsMenuOpen(true)}
          sx={{ display: isMenuOpen ? 'none' : 'inline-block' }}
        >
          <MenuIcon />
        </IconButton>
        <Box
          sx={{
            flex: 1,
            fontSize: 12.5,
            display: isMenuOpen ? 'flex' : 'none',
            alignItems: 'flex-start',
            flexDirection: 'column',
            position: 'fixed',
            left: 0,
            right: 0,
            top: 0,
            bottom: 0,
            bgcolor: 'black',
            zIndex: 10,
            overflowY: 'auto',
            pt: '1.5px',
            pb: 2,
          }}
        >
          <MenuIconMobile setIsMenuOpen={setIsMenuOpen} />
          <LinksMobile />
        </Box>
      </Box>
    </Box>
  )
}
