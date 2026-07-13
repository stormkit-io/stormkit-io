import { useContext } from 'react'
import { Routes, Route, RouteProps } from 'react-router-dom'
import { ThemeProvider } from '@mui/material/styles'
import { CacheProvider, EmotionCache } from '@emotion/react'
import CssBaseline from '@mui/material/CssBaseline'
import Box from '@mui/material/Box'
import theme from './mui-theme'
import Context from './context'

interface Props {
  routes: RouteProps[]
  cache: EmotionCache
}

export default function App({ routes, cache }: Props) {
  const { isHomePage } = useContext(Context)

  return (
    <CacheProvider value={cache}>
      <ThemeProvider theme={theme}>
        <CssBaseline />
        <Box
          bgcolor={!isHomePage ? '#0F092B' : ''}
          sx={{
            backgroundImage: isHomePage
              ? 'linear-gradient(111.4deg,rgba(7, 7, 9, 1) 6.5%,rgb(21, 19, 67) 93.2%);'
              : '',
            ':after': isHomePage
              ? {
                  content: '""',
                  position: 'absolute',
                  left: 0,
                  top: 0,
                  right: 0,
                  bottom: 0,
                  backgroundImage:
                    'radial-gradient(#ffffff 1px, transparent 1px), radial-gradient(#ffffff 1px, transparent 1px)',
                  backgroundSize: '50px 50px',
                  opacity: 0.15,
                  zIndex: 0,
                }
              : {},
          }}
        >
          <Routes>
            {routes.map((route) => (
              <Route key={route.path} {...route}></Route>
            ))}
          </Routes>
        </Box>
      </ThemeProvider>
    </CacheProvider>
  )
}
