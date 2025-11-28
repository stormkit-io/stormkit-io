import { useContext } from 'react'
import Box from '@mui/material/Box'
import HeaderDesktop from './HeaderDesktop'
import HeaderMobile from './HeaderMobile'
import Context from '~/context'

export default function Header() {
  const { isHomePage } = useContext(Context)

  return (
    <Box
      sx={{
        position: isHomePage ? 'sticky' : undefined,
        top: isHomePage ? -16 : 0,
        zIndex: 3,
      }}
    >
      <HeaderDesktop />
      <HeaderMobile />
    </Box>
  )
}
