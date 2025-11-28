import { useContext } from 'react'
import { useParams } from 'react-router'
import LinearProgress from '@mui/material/LinearProgress'
import Box from '@mui/material/Box'
import { purple, grey } from '@mui/material/colors'
import Header from '~/components/Header'
import Footer from '~/components/Footer'
import DocsNav from '~/components/DocsNav'
import ImageOverlay from '~/components/ImageOverlay'
import Error404 from '~/components/Error404'
import { withContent } from '~/helpers/markdown'
import { fetchData } from './_ssr'
import Context from '~/context'

// Required for SSR
export { fetchData } from './_ssr'

export default function DocTitle() {
  const params = useParams()
  const { url } = useContext(Context)
  const category = params.category || url?.split?.('/')?.[2] || ''
  const title = params.title || url?.split?.('/')?.[3] || ''
  const { content, navigation, loading } = withContent(fetchData, {
    defaultCategory: category,
    defaultTitle: title,
  })

  if (!content.__html && !loading) {
    return <Error404 />
  }

  return (
    <Box
      sx={{
        minHeight: '100vh',
        display: 'flex',
        flexDirection: 'column',
        color: 'primary.contrastText',
      }}
      className={params.category?.toLowerCase() || 'welcome'}
    >
      <Header />
      <Box sx={{ flex: 1, visibility: loading ? 'visible' : 'hidden' }}>
        <LinearProgress />
      </Box>
      <ImageOverlay content={content} navigation={navigation} />
      <Box
        sx={{
          display: 'flex',
          flexDirection: { xs: 'column', md: 'row' },
          mx: 'auto',
          flexGrow: 1,
          width: '100%',
          maxWidth: '1560px',
        }}
      >
        <DocsNav items={navigation} currentCategory={category} />
        <Box
          sx={{
            display: 'flex',
            flex: 1,
          }}
        >
          <Box
            sx={{
              p: { xs: 2, lg: 4 },
              pt: { xs: 2, lg: 2 },
              mx: 'auto',
              flex: 1,
              lineHeight: 2,
              '& summary': {
                cursor: 'pointer',
                bgcolor: 'rgba(0,0,0,0.3)',
                mb: 2,
                p: 2,
                ':hover': {
                  bgcolor: 'rgba(0,0,0,0.5)',
                },
                '& span:first-of-type': {
                  color: purple[400],
                  display: 'inline-block',
                  width: '4rem',
                  ml: 1,
                },
                '& span:last-child': {
                  color: grey[200],
                },
              },
            }}
            maxWidth="768px"
          >
            <div id="blog-content" dangerouslySetInnerHTML={content} />
          </Box>
        </Box>
      </Box>
      <Footer />
    </Box>
  )
}
