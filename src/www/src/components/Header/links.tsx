import Box from '@mui/material/Box'
import XIcon from '@mui/icons-material/X'
import LinkedInIcon from '@mui/icons-material/LinkedIn'
import YouTubeIcon from '@mui/icons-material/YouTube'
import CompareIcon from '@mui/icons-material/Compare'
import DiscordLogo from '~/assets/images/discord.svg'

const iconSx = { fontSize: 16, mr: 2 }

interface Section {
  title: string
  links: LinkProps[]
}

interface LinkProps {
  path: string
  id?: string
  subtext?: string
  icon?: React.ReactNode
  text?: React.ReactNode
  fill?: string
  children?: Section[]
  separator?: boolean
  maxWidth?: number
  position?: 'center' | 'left' | 'right'
}

const links: LinkProps[] = [
  {
    path: '/#resources',
    text: 'Resources',
    id: 'button-resources',
    maxWidth: 500,
    children: [
      {
        title: 'Company',
        links: [
          {
            path: '/about-us',
            text: 'About us',
          },
          {
            path: '/blog',
            text: 'Blog',
            subtext: 'Latest developments and announcements',
          },
          {
            path: '/blog/whats-new',
            text: 'Changelog',
            subtext: 'Latest changes on the product',
          },
          {
            path: '/policies/privacy',
            text: 'Privacy policy',
            subtext: 'Read our privacy policy',
          },
          {
            path: '/policies/terms',
            text: 'Terms',
            subtext: 'Read our terms and conditions',
          },
        ],
      },
      {
        title: 'Follow us',
        links: [
          {
            path: 'https://www.linkedin.com/company/stormkit',
            icon: <LinkedInIcon sx={iconSx} />,
            text: 'LinkedIn',
            fill: 'rgb(29, 155, 240)',
          },
          {
            path: 'https://www.youtube.com/@stormkit-io',
            text: 'YouTube',
            icon: <YouTubeIcon sx={iconSx} />,
            fill: 'rgb(255 0 0)',
          },
          {
            path: 'https://x.com/stormkitio',
            icon: <XIcon sx={iconSx} />,
            text: 'X',
            fill: 'white',
          },
          {
            path: 'https://discord.com/invite/6yQWhyY',
            text: 'Discord',
            icon: (
              <Box
                component="img"
                src={DiscordLogo}
                alt={'Discord'}
                sx={{
                  mr: 2,
                  display: 'inline-block',
                  width: '17px',
                  filter: 'brightness(0) invert(1)',
                }}
              />
            ),
          },
        ],
      },
      {
        title: 'Comparisons',
        links: [
          {
            path: '/vs-vercel',
            text: 'vs Vercel',
            icon: <CompareIcon sx={iconSx} />,
          },
          {
            path: '/vs-netlify',
            text: 'vs Netlify',
            icon: <CompareIcon sx={iconSx} />,
          },
        ],
      },
      {
        title: 'Migration guides',
        links: [
          {
            path: '/docs/migrations/vercel',
            text: 'Vercel',
            icon: <CompareIcon sx={iconSx} />,
          },
        ],
      },
    ],
  },
  {
    path: '/tutorials',
    text: 'Tutorials',
    maxWidth: 400,
    children: [
      {
        title: 'Deployment guides',
        links: [
          {
            text: 'Strapi CMS',
            path: '/tutorials/how-to-deploy-your-self-hosted-strapi-instance',
          },
          {
            text: 'Remix',
            path: '/tutorials/how-to-deploy-your-self-hosted-remix-app',
          },
        ],
      },
      {
        title: 'Self-Hosting',
        links: [
          {
            text: 'Hetzner Cloud',
            path: '/tutorials/how-to-self-host-stormkit-on-hetzner-cloud',
          },
          {
            text: 'Run Automated E2E Tests',
            path: '/tutorials/how-to-run-automated-e2-e-tests-with-stormkit-and-browserless',
          },
          {
            text: 'Local Development Guide',
            path: '/tutorials/setting-up-stormkit-with-cloudflare-tunnels',
          },
        ],
      },
    ],
  },
  { path: '/docs/welcome/getting-started', text: 'Docs' },

  { path: '/enterprise', text: 'Enterprise' },
  { path: '/#pricing', text: 'Pricing' },
]

export default links
