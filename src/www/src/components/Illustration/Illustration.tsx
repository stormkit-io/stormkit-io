import Box from '@mui/material/Box'
import Button from '@mui/material/Button'
import { grey } from '@mui/material/colors'
import Typography from '@mui/material/Typography'
import ArrowForwardIcon from '@mui/icons-material/ArrowForward'
import Deepbase from '~/assets/logos/third-party/deepbase.svg'
import Browser from '~/components/Browser'
import { features } from './features'

export default function Illustration() {
  const currentButton = features[0]

  return (
    <Box
      sx={{
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'space-between',
        position: 'relative',
        margin: '0 auto',
        flexDirection: 'column',
        mx: { xs: 2, md: 0 },
        maxWidth: '100%',
        overflow: 'hidden',
      }}
    >
      <Box maxWidth="xl" sx={{ textAlign: 'center', mx: 'auto', px: 2, mb: 4, width: '100%' }}>
        <Typography
          variant="h4"
          color="text.secondary"
          sx={{ mt: { xs: 10, md: 25 }, fontSize: 16 }}
        >
          Trusted by enterprises and teams
        </Typography>
        <Box
          sx={{
            mt: 4,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            flexWrap: 'wrap',
            gap: 2,
          }}
        >
          <Box
            component="img"
            alt="Elham"
            src="https://cdn.prod.website-files.com/64996137a1e0a71956ea90eb/64997d15979306f7e737a767_logo.png"
            sx={{ width: { xs: 60, sm: 80 }, maxWidth: '100%' }}
          />
          <Box
            sx={{
              display: 'flex',
              alignItems: 'center',
              bgcolor: 'rgba(255, 255, 255, 0.67)',
              p: 1,
              borderRadius: 2,
            }}
          >
            <Box
              component="img"
              src={Deepbase}
              alt="Deepbase"
              sx={{ width: { xs: 80, sm: 100 }, maxWidth: '100%' }}
            />
          </Box>
          <Box
            component="a"
            href="https://www.valpiccola.com"
            sx={{
              display: 'flex',
              alignItems: 'center',
              color: 'white',
              textDecoration: 'none',
            }}
            rel="noreferrer noopener"
          >
            <Box
              component="img"
              alt="Valpiccola"
              src="https://www.valpiccola.com/logo-small.png"
              sx={{ mr: 1, height: 32, maxWidth: '100%' }}
            />
            <Typography sx={{ fontWeight: 600 }}>Valpiccola</Typography>
          </Box>
        </Box>
      </Box>
      <Box
        sx={{
          border: `1px solid ${grey[900]}`,
          borderRadius: 2,
          mt: 4,
          position: 'relative',
          boxShadow:
            '0 0 20px 5px rgba(72, 80, 201, 0.4), 0 0 40px 20px rgba(72, 80, 201, 0.2)',
          maxWidth: '100%',
          width: '100%',
        }}
      >
        <Browser url="app.stormkit.io">
          <Box sx={{ p: 2, maxWidth: '100%', overflow: 'hidden' }}>
            <picture>
              <Box
                component="source"
                media="(max-width: 800px)"
                srcSet={`${currentButton.image}?size=800`}
              />
              <Box
                component="img"
                sx={{
                  width: '100%',
                  maxWidth: '100%',
                  height: 'auto',
                  display: 'block',
                  m: 0,
                  p: 0,
                  borderRadius: 1,
                }}
                src={`${currentButton.image}?size=1150`}
                alt="Deployments"
              />
            </picture>
          </Box>
        </Browser>
      </Box>
      <Box sx={{ maxWidth: 660, mt: 8, mx: 'auto', textAlign: 'center', px: 2, width: '100%' }}>
        <Typography sx={{ fontSize: { xs: 16, md: 24 }, mb: 2 }}>
          "Stormkit helped us cut deployment times from 24+ hours to 7 minutes,
          while cutting our infrastructure costs by 60%."
        </Typography>
        <Typography sx={{ fontSize: 16, mb: 2 }} color="text.secondary">
          Elham, Lead Developer
        </Typography>
        <Typography>
          <Button
            variant="contained"
            color="secondary"
            href="/blog/case-study-elham"
            data-umami-event="Case Study"
            sx={{ mt: 2, fontSize: 16 }}
          >
            Case study: Leading learning platform
            <ArrowForwardIcon sx={{ fontSize: 12, ml: 1 }} />
          </Button>
        </Typography>
      </Box>
    </Box>
  )
}