import { useState } from 'react'
import Box from '@mui/material/Box'
import Alert from '@mui/material/Alert'
import Typography from '@mui/material/Typography'
import { grey } from '@mui/material/colors'
import Footer from '~/components/Footer'
import Header from '~/components/Header'
import ContactForm from '~/components/ContactForm'

export default function Contact() {
  const [error, setError] = useState('')
  const [success, setSuccess] = useState(false)

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
      <Header />
      <Box
        sx={{
          flex: 1,
          display: 'flex',
          justifyContent: 'center',
          alignItems: 'center',
          px: { xs: 2, sm: 3, md: 4 },
          py: { xs: 4, md: 6 },
          maxWidth: '100%',
          overflow: 'hidden',
        }}
      >
        <Box
          sx={{
            mx: 'auto',
            maxWidth: 960,
            textAlign: 'center',
            width: '100%',
            minWidth: 0,
          }}
        >
          <Typography
            variant="h1"
            sx={{
              fontWeight: 'bold',
              my: { xs: 2, md: 4 },
              fontSize: { xs: 24, sm: 32, md: 40 },
              px: { xs: 1, sm: 2 },
              wordBreak: 'break-word',
            }}
          >
            Contact us
          </Typography>
          <Box sx={{ maxWidth: '100%', overflow: 'hidden' }}>
            <Box
              sx={{
                display: 'grid',
                gridTemplateColumns: { xs: '1fr', md: 'repeat(2, minmax(0, 1fr))' },
                border: '1px solid rgba(0, 0, 0, 0.3)',
                backgroundColor: 'rgba(0,0,0,0.3)',
                borderRadius: 2,
                p: { xs: 2, sm: 3, lg: 4 },
                mb: 4,
                gap: { xs: 2, md: 0 },
                maxWidth: '100%',
                overflow: 'hidden',
              }}
            >
              <Box 
                sx={{ 
                  textAlign: 'left', 
                  p: { xs: 1, sm: 2, lg: 4 },
                  minWidth: 0,
                  maxWidth: '100%',
                  overflow: 'hidden',
                }}
              >
                <Typography 
                  variant="h5" 
                  sx={{ 
                    mb: 2,
                    fontSize: { xs: 18, sm: 20, md: 24 },
                    wordBreak: 'break-word',
                  }}
                >
                  <Box component="span" sx={{ fontWeight: 'bold' }}>
                    Curious about what Stormkit can do for your organization?
                  </Box>
                </Typography>
                <Typography 
                  sx={{ 
                    color: grey[400],
                    fontSize: { xs: 14, sm: 15, md: 16 },
                    wordBreak: 'break-word',
                    lineHeight: 1.6,
                  }}
                >
                  Our team will tailor a demonstration to your specific business
                  needs, exploring how our platform can accelerate your
                  development workflows and deployment processes. We'll walk
                  through our enterprise features, discuss flexible pricing
                  options, and help you set up a trial environment.
                </Typography>
              </Box>
              <Box
                sx={{
                  minWidth: 0,
                  maxWidth: '100%',
                  overflow: 'hidden',
                }}
              >
                <ContactForm setSuccess={setSuccess} setError={setError} />
              </Box>
            </Box>
            {error && (
              <Alert 
                variant="filled" 
                color="error" 
                sx={{ 
                  mb: 4,
                  mx: { xs: 1, sm: 0 },
                  wordBreak: 'break-word',
                }}
              >
                {error}
              </Alert>
            )}
            {success && (
              <Alert 
                variant="filled" 
                color="success" 
                sx={{ 
                  mb: 4,
                  mx: { xs: 1, sm: 0 },
                  wordBreak: 'break-word',
                }}
              >
                Your contact request has been forwarded. You will be contacted
                soon on the provided email.
              </Alert>
            )}
          </Box>
        </Box>
      </Box>
      <Footer />
    </Box>
  )
}