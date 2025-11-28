import Box from '@mui/material/Box'
import Link from '@mui/material/Link'
import Typography from '@mui/material/Typography'
import XIcon from '@mui/icons-material/X'
import LinkedInIcon from '@mui/icons-material/LinkedIn'
import YouTubeIcon from '@mui/icons-material/YouTube'
import StormkitLogoCircle from '~/assets/logos/stormkit-logo-circle.svg'
import DiscordLogo from '~/assets/images/discord.svg'

interface Props {
  maxWidth?: string
}

let lastWidth: string = 'lg'

interface LinkProps {
  path: string
  text: string
  icon?: React.ReactNode
  external?: boolean
}

const links: Array<{ name: string; links: LinkProps[] }> = [
  {
    name: 'company',
    links: [
      { path: '/#pricing', text: 'Pricing' },
      { path: '/policies/privacy', text: 'Privacy policy' },
      { path: '/policies/terms', text: 'Terms' },
    ],
  },
  {
    name: 'comparisons',
    links: [
      { path: '/vs-vercel', text: 'vs Vercel' },
      { path: '/vs-netlify', text: 'vs Netlify' },
    ],
  },
  {
    name: 'resources',
    links: [
      { path: '/docs/welcome/getting-started', text: 'Docs' },
      { path: '/blog', text: 'Blog' },
      { path: '/blog/whats-new', text: "What's new?" },
      { path: '/docs/welcome/contributing', text: 'Contribute' },
      {
        path: 'https://status.stormkit.io/',
        text: 'Status',
        external: true,
      },
    ],
  },
  {
    name: 'connect with us',
    links: [
      {
        path: 'https://x.com/stormkitio',
        text: 'X',
        external: true,
        icon: <XIcon sx={{ mr: 1, fill: 'rgb(29, 155, 240)' }} />,
      },
      {
        path: 'https://www.linkedin.com/company/stormkit',
        text: 'LinkedIn',
        external: true,
        icon: <LinkedInIcon sx={{ mr: 1, fill: 'rgb(29, 155, 240)' }} />,
      },
      {
        path: 'https://www.youtube.com/@stormkit-io',
        text: 'YouTube',
        external: true,
        icon: <YouTubeIcon sx={{ mr: 1, fill: 'rgb(255 0 0)' }} />,
      },
      {
        path: 'https://discord.com/invite/6yQWhyY',
        text: 'Discord',
        icon: (
          <Box
            component="img"
            src={DiscordLogo}
            alt={'Discord'}
            sx={{ mr: 1, display: 'inline-block', width: '24px' }}
          />
        ),
      },
    ],
  },
]

export default function Footer({ maxWidth = lastWidth }: Props) {
  lastWidth = maxWidth

  return (
    <Box
      sx={{
        px: { xs: 1, md: 0 },
        borderTop: '1px solid rgba(0,0,0,0.3)',
        bgcolor: { xs: 'black', md: 'transparent' },
      }}
    >
      <Box
        sx={{
          m: 'auto',
          px: { xs: 2, md: 4 },
          display: 'flex',
          flexDirection: { xs: 'column', md: 'row' },
          alignItems: 'flex-start',
          justifyContent: 'space-between',
          maxWidth: '1560px',
        }}
      >
        <Box
          sx={{
            mt: 3,
            display: 'flex',
            alignItems: 'center',
          }}
        >
          <Link href="/" sx={{ display: 'block', width: 32, mr: 2 }}>
            <Box
              component="img"
              src={StormkitLogoCircle}
              alt="Stormkit Logo"
              width="100%"
            />
          </Link>
          <Typography variant="body2">
            {new Date().getFullYear()} Stormkit OÜ ©
          </Typography>
        </Box>
        <Box sx={{ flex: 1, ml: { xs: 0, md: 16 }, py: 4 }}>
          <Box
            sx={{
              display: 'flex',
              flexDirection: { xs: 'column', md: 'row' },
              justifyContent: 'space-between',
            }}
          >
            {links.map((group) => (
              <Box
                key={group.name}
                sx={{
                  minWidth: `${100 / links.length}%`,
                  display: 'flex',
                  flexDirection: 'column',
                  mb: { xs: 4, md: 0 },
                  ':last-child': {
                    mb: { xs: 0 },
                  },
                }}
              >
                <Typography
                  variant="h5"
                  sx={{
                    fontWeight: 600,
                    textTransform: 'uppercase',
                    fontSize: 12,
                    mb: 2,
                  }}
                >
                  {group.name}
                </Typography>
                {group.links.map((link) => (
                  <Link
                    key={link.path}
                    color="primary.contrastText"
                    target={link.external ? '_blank' : undefined}
                    rel={link.external ? 'noopener noreferrer' : undefined}
                    sx={{
                      mb: 0.5,
                      display: 'inline-flex',
                      alignItems: 'center',
                      textDecoration: 'none',
                      opacity: 0.7,
                      transition: 'opacity 0.25s ease-in-out',
                      ':hover': { opacity: 1 },
                    }}
                    href={link.path}
                  >
                    {link.icon}
                    {link.text}
                  </Link>
                ))}
              </Box>
            ))}
          </Box>
        </Box>
      </Box>
    </Box>
  )
}
