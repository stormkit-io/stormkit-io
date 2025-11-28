import Box, { BoxProps } from '@mui/material/Box'
import Header from '~/components/Header'
import Footer from '~/components/Footer'
import Illustration from '~/components/Illustration'
import Pricing from '~/components/Pricing'
import { useScrollToHash } from '~/helpers/scroll'
import Workflow from './_components/Workflow'
import Hero from './_components/Hero'
import Enterprise from './_components/Enterprise'
import DeployNow from './_components/DeployNow'
import FAQ from './_components/FAQ'

const Section = ({ maxWidth = 'xl', sx, children, id }: BoxProps) => {
  return (
    <Box
      id={id}
      maxWidth={maxWidth}
      sx={{
        mx: 'auto',
        width: '100%',
        mb: { xs: 10, md: 25 },
        zIndex: 1,
        position: 'relative',
        ...sx,
      }}
    >
      {children}
    </Box>
  )
}

export default function Home() {
  useScrollToHash()

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
          display: 'flex',
          flexDirection: 'column',
          justifyContent: 'center',
          alignItems: 'center',
          width: '100%',
          flex: 1,
          mx: 'auto',
          minHeight: { xs: '57vh', md: 'calc(100vh - 104px)' }, // 104 is the header height
          px: { xs: 2, md: 0 },
          zIndex: 2,
          opacity: 0,
          animation: 'fadeInUp 1s ease-out 0.25s forwards',
        }}
        maxWidth="xl"
      >
        <Hero />
      </Box>

      <Section id="feature-preview">
        <Illustration />
      </Section>

      <Section id="workflow">
        <Workflow />
      </Section>

      <Section id="enterprise">
        <Enterprise />
      </Section>

      <Section id="deploy-now">
        <DeployNow />
      </Section>

      <Section id="pricing">
        <Pricing />
      </Section>

      <Section id="faq">
        <FAQ />
      </Section>
      <Footer maxWidth="lg" />
    </Box>
  )
}
