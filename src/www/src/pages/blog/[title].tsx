import { useSearchParams } from 'react-router-dom'
import Box from '@mui/material/Box'
import Typography from '@mui/material/Typography'
import Link from '@mui/material/Link'
import Avatar from '@mui/material/Avatar'
import Header from '~/components/Header'
import Footer from '~/components/Footer'
import ImageOverlay from '~/components/ImageOverlay'
import ArticleProgress from '~/components/ArticleProgress'
import ArticleToc from '~/components/ArticleToc'
import { dateFormat } from '~/helpers/date'
import { withContent } from '~/helpers/markdown'
import { fetchData } from './_ssr'

// Required for SSR
export { fetchData } from './_ssr'

// Average adult reading speed for technical prose, rounded down to stay
// honest: a reader who finishes early is pleased, one who runs over is not.
const WORDS_PER_MINUTE = 220

function readingTime(html?: string): number {
  if (!html) {
    return 0
  }

  const words = html
    .replace(/<[^>]+>/g, ' ')
    .trim()
    .split(/\s+/).length

  return Math.max(1, Math.round(words / WORDS_PER_MINUTE))
}

export default function BlogContent() {
  const [searchParams] = useSearchParams()
  const { content, navigation } = withContent(fetchData)

  const { title, subtitle, description, date, author } =
    navigation.find((n) => n.active) || {}

  const isRaw = searchParams.get('raw') === 'true'
  const lede = subtitle || description
  const minutes = readingTime(content?.__html)

  return (
    <Box
      sx={{
        minHeight: '100vh',
        display: 'flex',
        flexDirection: 'column',
        bgcolor: 'background.default',
        color: 'primary.contrastText',
        maxWidth: '100%',
        // clip, not hidden: `hidden` makes this a scroll container, which
        // silently disables position: sticky on the contents index inside it.
        overflowX: 'clip',
      }}
    >
      {!isRaw && <ArticleProgress />}
      {!isRaw && <Header />}
      <ImageOverlay content={content} navigation={navigation} />

      <Box
        sx={{
          display: 'flex',
          justifyContent: 'center',
          flexGrow: 1,
          width: '100%',
          px: { xs: 2, sm: 3, md: 4 },
          pt: isRaw ? 0 : { xs: 6, md: 10 },
          pb: 8,
        }}
      >
        {/* The index is offset by its own width so the prose stays optically
            centred on the page rather than pushed right by the sidebar. */}
        <Box
          sx={{
            display: { xs: 'none', lg: 'block' },
            width: 232,
            flexShrink: 0,
          }}
        >
          {!isRaw && <ArticleToc content={content} />}
        </Box>

        <Box sx={{ width: '100%', maxWidth: '768px', minWidth: 0 }}>
          {!isRaw && (
            <Box
              component="header"
              sx={{
                position: 'relative',
                pb: { xs: 4, md: 6 },
                // A dot field instead of a light source: the earlier glow put
                // its brightest part over the prose and cost legibility. This
                // adds texture without adding luminance behind the text, and the
                // radial mask dissolves it before the first paragraph.
                // It overflows the header, so whatever is painted after it needs
                // its own stacking position to stay on top.
                '&::before': {
                  content: '""',
                  position: 'absolute',
                  top: '-25vh',
                  left: '50%',
                  transform: 'translateX(-50%)',
                  width: '100vw',
                  height: '90vh',
                  backgroundImage:
                    'radial-gradient(rgba(255,255,255,0.16) 1px, transparent 1px)',
                  backgroundSize: '22px 22px',
                  maskImage:
                    'radial-gradient(ellipse 46% 46% at 50% 34%, #000 0%, transparent 70%)',
                  WebkitMaskImage:
                    'radial-gradient(ellipse 46% 46% at 50% 34%, #000 0%, transparent 70%)',
                  pointerEvents: 'none',
                },
              }}
            >
              <Box sx={{ position: 'relative' }}>
                <Link
                  href="/blog"
                  sx={{
                    display: 'inline-block',
                    fontSize: 11,
                    fontWeight: 600,
                    letterSpacing: '0.14em',
                    textTransform: 'uppercase',
                    color: '#e2ae5a',
                    textDecoration: 'none',
                    mb: 2.5,
                    ':hover': { textDecoration: 'underline' },
                  }}
                >
                  ← Blog
                </Link>
                <Typography
                  variant="h1"
                  sx={{
                    fontSize: { xs: 30, sm: 38, md: 46 },
                    fontWeight: 600,
                    lineHeight: 1.12,
                    letterSpacing: '-0.025em',
                    mb: lede ? 2.5 : 3,
                    textWrap: 'balance',
                  }}
                >
                  {title}
                </Typography>
                {lede && (
                  <Typography
                    sx={{
                      fontSize: { xs: 16, md: 19 },
                      lineHeight: 1.6,
                      fontWeight: 400,
                      color: 'rgba(255,255,255,0.62)',
                      mb: 3.5,
                      maxWidth: '62ch',
                    }}
                  >
                    {lede}
                  </Typography>
                )}
                <Box
                  sx={{
                    display: 'flex',
                    alignItems: 'center',
                    gap: 1.5,
                    flexWrap: 'wrap',
                    fontSize: 13,
                    color: 'rgba(255,255,255,0.5)',
                  }}
                >
                  {author && (
                    <>
                      <Avatar
                        src={author.img}
                        alt={`${author.name} profile`}
                        sx={{ width: 30, height: 30 }}
                      />
                      <Box
                        component="span"
                        sx={{ color: 'rgba(255,255,255,0.8)' }}
                      >
                        {author.name}
                      </Box>
                      <Box component="span" sx={{ opacity: 0.35 }}>
                        ·
                      </Box>
                    </>
                  )}
                  {date && <Box component="span">{dateFormat(date)}</Box>}
                  {minutes > 0 && (
                    <>
                      <Box component="span" sx={{ opacity: 0.35 }}>
                        ·
                      </Box>
                      <Box component="span">{minutes} min read</Box>
                    </>
                  )}
                </Box>
              </Box>
            </Box>
          )}

          <Box
            component="article"
            sx={{
              position: 'relative',
              width: '100%',
              minWidth: 0,
              '& #blog-content': {
                maxWidth: '100%',
                wordBreak: 'break-word',
                '& img': { maxWidth: '100%', height: 'auto', display: 'block' },
                '& pre': { maxWidth: '100%', overflow: 'auto' },
                '& code': {
                  wordBreak: 'break-word',
                  overflowWrap: 'break-word',
                },
                '& table': {
                  maxWidth: '100%',
                  overflow: 'auto',
                  display: 'block',
                },
                '& video': { maxWidth: '100%', height: 'auto' },
              },
            }}
          >
            {isRaw && (
              <Typography variant="h1" sx={{ fontSize: 24, fontWeight: 600 }}>
                {title}
              </Typography>
            )}
            <div id="blog-content" dangerouslySetInnerHTML={content} />

            {author && !isRaw && (
              <Box
                sx={{
                  mt: 8,
                  p: 3,
                  display: 'flex',
                  alignItems: 'center',
                  gap: 2,
                  flexWrap: 'wrap',
                  borderRadius: 2,
                  border: '1px solid rgba(255,255,255,0.08)',
                  bgcolor: 'rgba(255,255,255,0.02)',
                }}
              >
                <Avatar
                  src={author.img}
                  alt={`${author.name} profile`}
                  sx={{ width: 52, height: 52 }}
                />
                <Box sx={{ minWidth: 0 }}>
                  <Typography sx={{ fontWeight: 600, lineHeight: 1.4 }}>
                    {author.name}
                  </Typography>
                  <Link
                    href={`https://x.com/${author.twitter.replace('@', '')}`}
                    target="_blank"
                    rel="noreferrer noopener"
                    sx={{
                      fontSize: 13,
                      color: '#e2ae5a',
                      textDecoration: 'none',
                      ':hover': { textDecoration: 'underline' },
                    }}
                  >
                    {author.twitter}
                  </Link>
                </Box>
              </Box>
            )}
          </Box>
        </Box>

        {/* Balances the sidebar on the opposite side so the article column
            stays centred; empty on anything narrower than lg. */}
        <Box
          aria-hidden
          sx={{
            display: { xs: 'none', lg: 'block' },
            width: 232,
            flexShrink: 0,
          }}
        />
      </Box>
      {!isRaw && <Footer maxWidth="lg" />}
    </Box>
  )
}
