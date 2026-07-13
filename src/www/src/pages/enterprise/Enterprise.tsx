import { useState } from 'react'
import Box from '@mui/material/Box'
import Alert from '@mui/material/Alert'
import Typography from '@mui/material/Typography'
import { grey } from '@mui/material/colors'
import CheckCircleIcon from '@mui/icons-material/CheckCircle'
import ContactForm from '~/components/ContactForm'
import Footer from '~/components/Footer'
import Header from '~/components/Header'

export const fetchData: FetchDataFunc = async () => {
  return {
    head: {
      title: 'Enterprise-grade Solutions for Modern Teams',
      description:
        'Get a custom demo of our enterprise-grade development platform. Dedicated support, enhanced security & scalable infrastructure for modern teams.',
    },
    context: {},
  }
}

const benefits = [
  {
    title: 'Dedicated Account Management',
    desc: 'Get personalized support from a dedicated account manager who understands your business needs and technical requirements.',
  },
  {
    title: 'Custom SLAs and Priority Support',
    desc: 'Get personalized support from a dedicated account manager who understands your business needs and technical requirements.',
  },
  {
    title: 'Scalable Infrastructure',
    desc: 'Our enterprise solution scales with your organization, handling your growing development and deployment needs.',
  },
  {
    title: 'Custom features',
    desc: "We build custom features tailored to your organization's specific needs. Our team works closely with you to understand your unique workflows and develop solutions that seamlessly integrate with your existing systems.",
  },
]

export default function Enterprise() {
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
              fontSize: { xs: 24, sm: 28, md: 32 },
              px: { xs: 1, sm: 2 },
              wordBreak: 'break-word',
            }}
          >
            Enterprise-grade Solutions for Modern Teams
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
                    Transform your development workflow with our enterprise
                    platform
                  </Box>
                </Typography>
                <Typography 
                  sx={{ 
                    color: grey[400], 
                    mb: 2,
                    fontSize: { xs: 14, sm: 15, md: 16 },
                    wordBreak: 'break-word',
                  }}
                >
                  We will create a customized plan for your organization,
                  addressing your unique challenges and helping you achieve your
                  business goals.
                </Typography>
                {benefits.map((b) => (
                  <Box 
                    key={b.title} 
                    sx={{ 
                      mb: { xs: 3, md: 2 },
                      minWidth: 0,
                      maxWidth: '100%',
                    }}
                  >
                    <Typography 
                      variant="h6" 
                      sx={{ 
                        mb: 1,
                        fontSize: { xs: 16, sm: 18, md: 20 },
                      }}
                    >
                      <Box
                        component="span"
                        sx={{
                          fontWeight: 'bold',
                          display: 'flex',
                          alignItems: 'center',
                          wordBreak: 'break-word',
                        }}
                      >
                        <CheckCircleIcon
                          sx={{ 
                            mr: 1, 
                            fontSize: { xs: 14, md: 16 }, 
                            fill: '#78193B',
                            flexShrink: 0,
                          }}
                        />
                        <Box component="span" sx={{ minWidth: 0 }}>
                          {b.title}
                        </Box>
                      </Box>
                    </Typography>
                    <Typography 
                      sx={{ 
                        color: grey[400],
                        fontSize: { xs: 13, sm: 14, md: 15 },
                        wordBreak: 'break-word',
                        lineHeight: 1.6,
                      }}
                    >
                      {b.desc}
                    </Typography>
                  </Box>
                ))}
              </Box>
              <Box
                sx={{
                  minWidth: 0,
                  maxWidth: '100%',
                  overflow: 'hidden',
                }}
              >
                <ContactForm setError={setError} setSuccess={setSuccess} />
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