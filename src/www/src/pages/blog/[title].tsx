import { useSearchParams } from 'react-router-dom'
import Box from '@mui/material/Box'
import Typography from '@mui/material/Typography'
import Link from '@mui/material/Link'
import Avatar from '@mui/material/Avatar'
import Header from '~/components/Header'
import Footer from '~/components/Footer'
import ImageOverlay from '~/components/ImageOverlay'
import { dateFormat } from '~/helpers/date'
import { withContent } from '~/helpers/markdown'
import { fetchData } from './_ssr'

// Required for SSR
export { fetchData } from './_ssr'

export default function BlogContent() {
  const [searchParams] = useSearchParams()
  const { content, navigation } = withContent(fetchData)

  const { title, subtitle, date, author } =
    navigation.find((n) => n.active) || {}

  const isRaw = searchParams.get('raw') === 'true'

  return (
    <Box
      sx={{
        minHeight: '100vh',
        display: 'flex',
        flexDirection: 'column',
        bgcolor: 'background.default',
        color: 'primary.contrastText',
        maxWidth: '100%',
        overflow: 'hidden',
      }}
    >
      {!isRaw && <Header />}
      <ImageOverlay content={content} navigation={navigation} />
      <Box
        maxWidth="none"
        sx={{
          display: 'flex',
          flexDirection: { xs: 'column', md: 'row' },
          mx: 'auto',
          flexGrow: 1,
          width: '100%',
          maxWidth: '100%',
          overflow: 'hidden',
        }}
      >
        <Box
          sx={{
            display: 'flex',
            flex: 1,
            width: '100%',
            maxWidth: '100%',
            overflow: 'hidden',
          }}
        >
          <Box
            sx={{
              p: { xs: 2, lg: 4 },
              pt: { xs: 2, lg: 2 },
              mx: { xs: 0, md: 'auto' },
              px: { xs: 2, sm: 3, md: 4 },
              flex: 1,
              bgcolor: 'rgba(0,0,0,0.05)',
              lineHeight: 2,
              mt: isRaw ? 0 : 4,
              width: '100%',
              maxWidth: { xs: '100%', md: '768px' },
              minWidth: 0,
              overflow: 'hidden',
              '& #blog-content': {
                maxWidth: '100%',
                overflow: 'hidden',
                wordBreak: 'break-word',
                '& img': {
                  maxWidth: '100%',
                  height: 'auto',
                  display: 'block',
                },
                '& pre': {
                  maxWidth: '100%',
                  overflow: 'auto',
                  WebkitOverflowScrolling: 'touch',
                },
                '& code': {
                  wordBreak: 'break-word',
                  overflowWrap: 'break-word',
                },
                '& table': {
                  maxWidth: '100%',
                  overflow: 'auto',
                  display: 'block',
                },
                '& iframe, & video': {
                  maxWidth: '100%',
                  height: 'auto',
                },
              },
            }}
          >
            <Typography
              variant="h1"
              sx={{
                fontSize: { xs: 20, sm: 22, md: 24 },
                fontWeight: 600,
                mt: isRaw ? 0 : 4,
                wordBreak: 'break-word',
              }}
            >
              {title}
            </Typography>
            {subtitle && (
              <Typography
                variant="h2"
                sx={{
                  fontSize: { xs: 14, sm: 15, md: 16 },
                  fontWeight: 600,
                  my: 1,
                  opacity: 0.7,
                  wordBreak: 'break-word',
                }}
              >
                {subtitle}
              </Typography>
            )}
            {date && (
              <Typography
                sx={{
                  opacity: 0.7,
                  fontSize: { xs: 12, sm: 13 },
                  mb: 4,
                }}
              >
                {dateFormat(date)}
              </Typography>
            )}
            <div id="blog-content" dangerouslySetInnerHTML={content} />
            {author && (
              <Box
                sx={{
                  pt: 2,
                  display: 'flex',
                  alignItems: 'center',
                  borderTop: '1px solid rgba(255,255,255,0.1)',
                  flexWrap: 'wrap',
                  wordBreak: 'break-word',
                }}
              >
                <Link
                  href={`https://x.com/${author.twitter.replace('@', '')}`}
                  target="_blank"
                  rel="noreferrer noopener"
                  sx={{
                    display: 'inline-flex',
                    alignItems: 'center',
                    color: 'white',
                    textDecoration: 'none',
                    ':hover': { textDecoration: 'underline' },
                  }}
                >
                  <Avatar
                    src={author.img}
                    sx={{ mr: 1, flexShrink: 0 }}
                    alt={`${author.name} profile`}
                  />
                  {author.name}
                </Link>
              </Box>
            )}
          </Box>
        </Box>
      </Box>
      {!isRaw && <Footer maxWidth="lg" />}
    </Box>
  )
}
