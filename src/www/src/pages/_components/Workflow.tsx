import Box, { BoxProps } from '@mui/material/Box'
import Typography from '@mui/material/Typography'
import Button from '@mui/material/Button'
import CheckIcon from '@mui/icons-material/Check'
import SettingsIcon from '@mui/icons-material/Settings'
import { green, grey } from '@mui/material/colors'
import GithubLogo from '~/assets/logos/github-logo.svg'
import GitlabLogo from '~/assets/logos/gitlab-logo.svg'
import BitbucketLogo from '~/assets/logos/bitbucket-logo.svg'
import StyledBox from '~/components/StyledBox'
import Browser from '~/components/Browser'

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

const buttons = [
  { logo: GithubLogo, text: 'Github' },
  { logo: GitlabLogo, text: 'GitLab' },
  { logo: BitbucketLogo, text: 'BitBucket' },
]

const Published = ({ sx }: BoxProps) => (
  <Box
    sx={{
      fontSize: 12,
      py: 0.25,
      px: 2,
      ml: 2,
      borderRadius: 2,
      bgcolor: green[900],
    }}
  >
    published
  </Box>
)

export default function Workflow() {
  return (
    <Box
      sx={{
        display: 'grid',
        gridTemplateColumns: { xs: '1fr', md: '1fr 1fr' },
        gap: 0,
      }}
    >
      {/* Left Column */}
      <Box sx={{ borderRight: { md: '1px solid rgba(255, 255, 255, 0.1)' } }}>
        <Box sx={{ p: { xs: 4, md: 10 } }}>
          <Title>Connect your repository</Title>
          <Typography variant="subtitle2" color="text.secondary">
            Link your repository to get started with automated deployments
          </Typography>
          <StyledBox
            sx={{
              textAlign: 'center',
              userSelect: 'none',
              display: 'flex',
              flexWrap: 'wrap',
              justifyContent: 'center',
              gap: { xs: 2, sm: 2, md: 1 },
            }}
          >
            {buttons.map((b) => (
              <Button
                key={b.text}
                variant="outlined"
                size="small"
                href="https://www.stormkit.io/docs/self-hosting/authentication"
                sx={{
                  border: '1px solid rgba(255, 255, 255, 0.15)',
                  minWidth: { xs: '45%', sm: 'auto' },
                }}
              >
                <Box
                  component="img"
                  src={b.logo}
                  alt={`${b.text} Integration`}
                  sx={{ display: 'inline-block', mr: 1 }}
                />
                {b.text}
              </Button>
            ))}
          </StyledBox>
        </Box>
        <Box
          sx={{
            p: { xs: 4, md: 10 },
            borderTop: { md: '1px solid rgba(255, 255, 255, 0.1)' },
            textAlign: 'left',
          }}
        >
          <Title>Enterprise-grade features</Title>
          <Typography variant="subtitle2" color="text.secondary">
            All the tools you need for ensuring code quality
          </Typography>
          <Box sx={{ mt: 6 }}>
            {[
              {
                title: 'Audit logs',
                description: 'Track changes of your team with detailed logs',
              },
              {
                title: 'Automated E2E testing',
                description: 'Run automated scripts after deployments',
              },
              {
                title: 'Custom TLS certificates',
                description: 'Use your own TLS certificates for custom domains',
              },
            ].map(({ title, description }, index) => (
              <StyledBox
                key={title}
                sx={{
                  position: 'relative',
                  mt: index === 0 ? 0 : -2,
                  left:
                    index === 0
                      ? { xs: -16, md: -32 }
                      : index === 1
                      ? 0
                      : { xs: 16, md: 32 },
                }}
              >
                <Box sx={{ display: 'flex', alignItems: 'center', mb: 1 }}>
                  <CheckIcon sx={{ mr: 2, fontSize: 12, color: grey[600] }} />
                  {title}
                </Box>
                <Typography color="text.secondary" fontSize="small">
                  {description}
                </Typography>
              </StyledBox>
            ))}
          </Box>
        </Box>

        <Box
          sx={{
            p: { xs: 4, md: 10 },
            borderTop: { md: '1px solid rgba(255, 255, 255, 0.1)' },
          }}
        >
          <Title>Zero downtime rolling updates</Title>
          <Typography variant="subtitle2" color="text.secondary">
            Deploy updates seamlessly without interrupting your users
          </Typography>
          <Box sx={{ mt: 6 }}>
            <StyledBox sx={{ mt: 2, p: 2 }}>
              <Box sx={{ display: 'flex', alignItems: 'center' }}>
                <SettingsIcon
                  sx={{
                    mr: 1,
                    fontSize: 14,
                    color: grey[600],
                    animation: 'rotate 2s linear infinite',
                  }}
                />
                feat: make table responsive
              </Box>
            </StyledBox>
            <StyledBox sx={{ mt: 2, p: 2 }}>
              <Box sx={{ display: 'flex', alignItems: 'center' }}>
                <CheckIcon
                  color="success"
                  sx={{
                    mr: 1,
                    fontSize: 14,
                  }}
                />
                fix: text overflow
                <Published />
              </Box>
            </StyledBox>
          </Box>
        </Box>
      </Box>

      {/* Right Column */}
      <Box>
        <Box sx={{ p: { xs: 4, md: 10 } }}>
          <Title>Git push to deploy</Title>
          <Typography variant="subtitle2" color="text.secondary">
            Automatically deploy your changes when you push to your repository
          </Typography>
          <Box
            sx={{
              mt: 6,
              background:
                'linear-gradient(135deg, rgba(61, 60, 64, 0.3) 0%, #0c0c1f 100%)',
              bgcolor: 'rgba(0, 0, 0, 0.15)',
              border: '1px solid rgba(255, 255, 255, 0.1)',
              borderRight: 'none',
              borderBottom: 'none',
              borderRadius: 2,
            }}
          >
            <Browser
              url="mywebsite.com"
              pageBgColor="linear-gradient(135deg, rgba(61, 60, 64, 0.3) 0%, #0c0c1f 100%)"
            >
              <Box sx={{ p: { xs: 2, sm: 3, md: 4 }, textAlign: 'center', fontSize: { xs: '12px', sm: '13px', md: '14px' } }}>
                Deployed with 💜 by Stormkit
                <Box sx={{ mt: { xs: 2, md: 4 } }}>
                  <Box component="ul" sx={{ listStyle: 'none', p: 0, m: 0 }}>
                    {[
                      'Unique URLs for each deployment',
                      'Automatic TLS',
                      'Instant rollbacks',
                    ].map((item, index) => (
                      <Box
                        key={index}
                        component="li"
                        sx={{
                          background:
                            'linear-gradient(135deg, rgba(61, 60, 64, 0.3) 0%, #0c0c1f 100%)',
                          display: 'flex',
                          alignItems: 'center',
                          mb: 1,
                          p: { xs: 0.75, sm: 1 },
                          borderRadius: 4,
                          fontSize: { xs: '11px', sm: '12px', md: '14px' },
                          minWidth: 0,
                          wordBreak: 'break-word',
                        }}
                      >
                        <CheckIcon
                          sx={{ fontSize: { xs: 10, sm: 11, md: 12 }, mr: { xs: 1, sm: 1.5, md: 2 }, flexShrink: 0 }}
                          color="success"
                        />
                        <Box component="span" sx={{ minWidth: 0, flex: 1 }}>
                          {item}
                        </Box>
                      </Box>
                    ))}
                  </Box>
                </Box>
              </Box>
            </Browser>
          </Box>
        </Box>
        <Box
          sx={{
            p: { xs: 4, md: 10 },
            borderTop: { md: '1px solid rgba(255, 255, 255, 0.1)' },
          }}
        >
          <Title>Monitor your application</Title>
          <Typography variant="subtitle2" color="text.secondary">
            Track performance, errors, and user analytics in real-time
          </Typography>
          <StyledBox sx={{ mt: 6, position: 'relative', height: 300 }}>
            <Box
              sx={{ display: 'flex', justifyContent: 'space-between', mb: 2 }}
            >
              <Typography variant="caption" color="text.secondary">
                Number of visitors
              </Typography>
              <Typography variant="caption" color="success.main">
                ↗ 23% last month
              </Typography>
            </Box>
            <Box sx={{ position: 'relative', height: 220 }}>
              <svg width="100%" height="100%" viewBox="0 0 300 100">
                <defs>
                  <linearGradient
                    id="lineGradient"
                    x1="0%"
                    y1="0%"
                    x2="100%"
                    y2="0%"
                  >
                    <stop offset="0%" stopColor="rgba(76, 175, 80, 0.8)" />
                    <stop offset="100%" stopColor="rgba(33, 150, 243, 0.8)" />
                  </linearGradient>
                </defs>
                <polyline
                  fill="none"
                  stroke="url(#lineGradient)"
                  strokeWidth="2"
                  points="0,80 30,75 60,70 90,60 120,55 150,45 180,40 210,35 240,30 270,25 300,20"
                />
                <circle cx="297" cy="20" r="3" fill="#4CAF50" />
              </svg>
            </Box>
          </StyledBox>
        </Box>
      </Box>
    </Box>
  )
}
