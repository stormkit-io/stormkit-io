import Box from '@mui/material/Box'
import Typography from '@mui/material/Typography'
import Header from '~/components/Header'
import Footer from '~/components/Footer'

export default function Home() {
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
        sx={{
          flex: 1,
          m: 'auto',
          px: { xs: 2, md: 0 },
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
        }}
        maxWidth="lg"
      >
        <Box sx={{ textAlign: 'center' }}>
          <Typography
            variant="h1"
            color="secondary"
            sx={{ fontWeight: 'bold' }}
          >
            4 oh 4
          </Typography>
          <Typography variant="h4" sx={{ mt: 4 }}>
            There is nothing under this link
          </Typography>
        </Box>
      </Box>
      <Footer />
    </Box>
  )
}
