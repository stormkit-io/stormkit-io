import Box from '@mui/material/Box'
import Link from '@mui/material/Link'
import IconButton from '@mui/material/IconButton'
import MenuIcon from '@mui/icons-material/Menu'
import StormkitLogoText from '~/assets/logos/stormkit-logo-text-h--white.svg'

export default function MobileMenuDesktop() {
  return (
    <Box sx={{ display: 'flex' }}>
      <Box sx={{ flex: 1 }}>
        <Link href="/" sx={{ display: 'block', width: 130, lineHeight: 1 }}>
          <Box
            component="img"
            src={StormkitLogoText}
            alt="Stormkit Logo"
            width="100%"
          />
        </Link>
      </Box>
      <IconButton sx={{ display: { md: 'none' } }}>
        <MenuIcon />
      </IconButton>
    </Box>
  )
}
