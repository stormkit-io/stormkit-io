import Typography from '@mui/material/Typography'
import Box from '@mui/material/Box'
import Link from '@mui/material/Link'
import ChevronRightIcon from '@mui/icons-material/ChevronRight'
import Header from '~/components/Header'
import Footer from '~/components/Footer'
import { withContent } from '~/helpers/markdown'
import { fetchData } from './_ssr'
import { dateFormat } from '~/helpers/date'

// Required for SSR
export { fetchData } from './_ssr'

export default function BlogContent() {
  const { content, navigation } = withContent(fetchData)

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
      <Box
        maxWidth="lg"
        sx={{
          mx: 'auto',
          flexGrow: 1,
          width: '100%',
          pt: 4,
        }}
      >
        {navigation?.map((nav) => (
          <Box
            key={nav.path}
            sx={{
              p: { xs: 2, lg: 4 },
              mx: 'auto',
              mb: 4,
              bgcolor: 'rgba(0,0,0,0.25)',
            }}
            maxWidth="768px"
          >
            {nav.date && (
              <Typography sx={{ mb: 1, fontSize: 13 }}>
                {dateFormat(nav.date)}
              </Typography>
            )}
            <Typography
              variant="h5"
              sx={{
                fontWeight: 600,
                fontSize: 20,
                display: 'flex',
                justifyContent: 'space-between',
              }}
            >
              <Link
                href={nav.path}
                sx={{
                  color: 'white',
                  textDecoration: 'none',
                  ':hover': { textDecoration: 'underline' },
                }}
              >
                {nav.title}
              </Link>
            </Typography>
            {nav.subtitle && (
              <Typography
                variant="h6"
                sx={{ fontSize: 14, mt: 0, opacity: 0.7 }}
              >
                {nav.subtitle}
              </Typography>
            )}
            <Typography variant="body1" sx={{ opacity: 0.7, mt: 1 }}>
              {nav.description}
            </Typography>
            <Box sx={{ textAlign: 'right', mt: 2 }}>
              <Link
                href={nav.path}
                sx={{
                  textDecoration: 'none',
                  display: 'inline-flex',
                  alignItems: 'center',
                  color: 'white',
                  ':hover': {
                    textDecoration: 'underline',
                  },
                }}
              >
                Read more <ChevronRightIcon />
              </Link>
            </Box>
          </Box>
        ))}
      </Box>
      <Footer maxWidth="lg" />
    </Box>
  )
}
