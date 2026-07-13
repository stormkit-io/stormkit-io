import { Typography } from '@mui/material'
import Box from '@mui/material/Box'
import StyledBox from '~/components/StyledBox'
import Astro from '~/assets/logos/third-party/astro.svg'
import Nuxt from '~/assets/logos/third-party/nuxt.svg'
import Next from '~/assets/logos/third-party/next.svg'
import Svelte from '~/assets/logos/third-party/svelte.svg'
import NodeJs from '~/assets/logos/third-party/nodejs.svg'
import Javascript from '~/assets/logos/third-party/js.svg'
import FastAPI from '~/assets/logos/third-party/fastapi.svg'
import Tanstack from '~/assets/logos/third-party/tanstack.png'
import Go from '~/assets/logos/third-party/go.svg'
import React from 'react'

const logos = [
  { logo: NodeJs, text: 'Node.js' },
  { logo: Javascript, text: 'JavaScript' },
  { logo: Astro, text: 'Astro' },
  { logo: Nuxt, text: 'Nuxt' },
  { logo: Next, text: 'Next' },
  { logo: Svelte, text: 'Svelte' },
  { logo: Tanstack, text: 'Tanstack' },
  { logo: FastAPI, text: 'FastAPI' },
  { logo: Go, text: 'Go' },
]

interface Props {
  children: React.ReactNode
}

const Title = ({ children }: Props) => (
  <Typography
    variant="h3"
    sx={{
      fontWeight: 600,
      fontSize: { xs: 24, md: 28 },
    }}
  >
    {children}
  </Typography>
)

const numbers = [
  {
    number: '600 millions',
    description: 'Requests served across all applications',
  },
  { number: '99.9%', description: 'Uptime reliability over the last 365 days' },
]

export default function DeployNow() {
  return (
    <Box
      sx={{
        display: 'flex',
        alignItems: 'center',
        flexDirection: 'column',
        p: { xs: 2, md: 8 },
        mx: 2,
        background:
          'linear-gradient(135deg, rgba(59, 130, 246, 0.1) 0%, rgba(147, 51, 234, 0.1) 100%)',
        backgroundImage: `url("data:image/svg+xml,%3csvg width='100%25' height='100%25' xmlns='http://www.w3.org/2000/svg'%3e%3crect width='100%25' height='100%25' fill='none' rx='11' ry='11' stroke='%23ffffff' stroke-width='1' stroke-dasharray='6%2c 14' stroke-dashoffset='100' stroke-linecap='round'/%3e%3c/svg%3e")`,
      }}
    >
      <Box>
        <Typography
          variant="h2"
          sx={{
            fontSize: { xs: 32, md: 48 },
            fontWeight: 600,
            mb: { xs: 2, md: 0 },
          }}
        >
          Deploy now your first application
        </Typography>
        <Typography
          variant="h3"
          sx={{
            fontSize: 18,
            textAlign: { md: 'center' },
            maxWidth: { md: 800 },
          }}
          color="text.secondary"
        >
          Self-host or deploy to our Cloud - supporting a wide range of
          frameworks and languages.
        </Typography>
      </Box>
      <Box
        sx={{
          display: 'flex',
          gap: 2,
          justifyContent: 'center',
          alignItems: 'stretch',
          my: 4,
          flexWrap: 'wrap',
        }}
      >
        {logos.map(({ logo, text }) => (
          <StyledBox
            key={text}
            sx={{
              p: 2,
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
            }}
          >
            <Box
              component="img"
              src={logo}
              alt={text}
              className="logo-img"
              sx={{ width: 32 }}
            />
          </StyledBox>
        ))}
      </Box>
      <Typography
        variant="h3"
        sx={{ fontSize: 18, textAlign: 'center', maxWidth: 800 }}
        color="text.secondary"
      >
        and many more...
      </Typography>
      <Box
        sx={{
          display: 'grid',
          gridTemplateColumns: { xs: 'repeat(1)', md: 'repeat(2, 1fr)' },
          gap: 4,
          mt: 8,
        }}
      >
        {[
          {
            title: 'No vendor lock-in',
            desc: `Stormkit is built on open standards and respects your freedom to choose. Export your data anytime, deploy anywhere, and never worry about being trapped in a proprietary ecosystem.`,
          },
          {
            title: 'Competitive pricing',
            desc: 'Our pricing model is transparent. We offer a few pricing tiers, with clear limits and no add-ons.',
          },
          {
            title: 'Strong Feature Set',
            desc: 'Stormkit is equipped with a powerful array of tools, ensuring that developers have everything they require to build, deploy, and scale applications seamlessly.',
          },
          {
            title: 'Direct support',
            desc: 'Get direct support from our engineering team to prepare your infrastructure. Our experts work with you to optimize performance, security, and scalability for your specific needs.',
          },
        ].map(({ title, desc }) => (
          <StyledBox key={title} sx={{ m: 0 }}>
            <Title>{title}</Title>
            <Typography variant="subtitle2" color="text.secondary">
              {desc}
            </Typography>
          </StyledBox>
        ))}
      </Box>
      <Box
        sx={{
          display: 'grid',
          gridTemplateColumns: { xs: '1fr', md: 'repeat(2, 1fr)' },
          gap: 4,
          mt: 8,
        }}
      >
        {numbers.map(({ number, description }) => (
          <Box
            key={number}
            color="text.secondary"
            sx={{
              p: 2,
              border: '1px solid',
              borderColor: 'primary.main',
              position: 'relative',
              bgcolor: 'rgba(0, 0, 0, 0.15)',
              borderRadius: 10,
              textAlign: 'center',
              boxShadow: 6,
            }}
          >
            <Typography
              sx={{ fontSize: 24, fontWeight: 600, mb: 1, color: 'white' }}
            >
              {number}
            </Typography>
            {description}
          </Box>
        ))}
      </Box>
    </Box>
  )
}
