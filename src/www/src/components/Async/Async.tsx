import React from 'react'
import Box from '@mui/material/Box'
import LinearProgress from '@mui/material/LinearProgress'
import Header from '~/components/Header'
import { matchPath } from 'react-router'

const Async = (
  path: string,
  dynamicImport: () => Promise<{ default: React.FC }>
): React.ReactNode => {
  const Component = React.lazy(dynamicImport)
  const isMatch = matchPath(path, window.location.pathname)

  // This is required to prevent React complaining from
  // client/server hydration mismatch. It should render
  // sync until the app is mounted completely. isMatch
  // is also required to render only matching routes - others
  // should be suspended.
  if (isMatch) {
    return <Component />
  }

  return (
    <React.Suspense
      fallback={
        <Box sx={{ bgcolor: '#0F092B', minHeight: '100vh' }}>
          <Header />
          <LinearProgress />
        </Box>
      }
    >
      <Component />
    </React.Suspense>
  )
}

export default Async
