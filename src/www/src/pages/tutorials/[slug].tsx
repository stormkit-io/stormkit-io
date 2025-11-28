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

export default function TutorialContent() {
  const { content, navigation } = withContent(fetchData)

  const { title, subtitle, date, author } =
    navigation.find((n) => n.active) || {}

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
      <ImageOverlay content={content} navigation={navigation} />
      <Box
        maxWidth="none"
        sx={{
          display: 'flex',
          flexDirection: { xs: 'column', md: 'row' },
          mx: 'auto',
          flexGrow: 1,
          width: '100%',
        }}
      >
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
              bgcolor: 'rgba(0,0,0,0.05)',
              lineHeight: 2,
              mt: 4,
            }}
            maxWidth="768px"
          >
            <Typography
              variant="h1"
              sx={{ fontSize: 24, fontWeight: 600, mt: 4 }}
            >
              {title}
            </Typography>
            {subtitle && (
              <Typography
                variant="h2"
                sx={{ fontSize: 16, fontWeight: 600, my: 1, opacity: 0.7 }}
              >
                {subtitle}
              </Typography>
            )}
            {date && (
              <Typography sx={{ opacity: 0.7, fontSize: 13, mb: 4 }}>
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
                    ':hover': { textDecoration: 'underline' },
                  }}
                >
                  <Avatar
                    src={author.img}
                    sx={{ mr: 1 }}
                    alt={`${author.name} profile`}
                  />
                  {author.name}
                </Link>
              </Box>
            )}
          </Box>
        </Box>
      </Box>
      <Footer maxWidth="lg" />
    </Box>
  )
}
