import Typography from '@mui/material/Typography'
import Box from '@mui/material/Box'
import StyledBox from '~/components/StyledBox'
import CloudIcon from '@mui/icons-material/Cloud'
import SecurityIcon from '@mui/icons-material/Security'
import ScaleIcon from '@mui/icons-material/TrendingUp'
import SettingsIcon from '@mui/icons-material/Settings'

export default function Enterprise() {
  return (
    <Box>
      <Box
        sx={{
          display: 'flex',
          alignItems: 'center',
          flexDirection: 'column',
          textAlign: 'center',
        }}
      >
        <Typography
          variant="h2"
          sx={{
            fontWeight: 600,
            fontSize: { xs: 24, md: 48 },
          }}
        >
          Bring your own Cloud
        </Typography>
        <Typography
          variant="h3"
          color="text.secondary"
          sx={{ fontSize: { xs: 16, md: 20, maxWidth: 600 } }}
        >
          Stormkit allows you to run your deployments on your own
          infrastructure, whether it's on-premises or in the cloud.
        </Typography>
      </Box>

      <StyledBox
        sx={{
          display: 'grid',
          gridTemplateColumns: { xs: '1fr', md: 'repeat(4, 1fr)' },
          gap: 4,
          mx: { xs: 2, md: 0 },
        }}
      >
        {[
          {
            icon: CloudIcon,
            title: 'Cloud-Native Integrations',
            description:
              'Stormkit is designed to seamlessly integrate with your existing cloud infrastructure, allowing you to leverage the tools and services you already use.',
          },
          {
            icon: SecurityIcon,
            title: 'Privacy-first',
            description:
              'Your data remains private and secure, as Stormkit operates on your own infrastructure, ensuring compliance with your security policies.',
          },
          {
            icon: ScaleIcon,
            title: 'Scalable Architecture',
            description:
              'Stormkit’s architecture is built to scale with your needs, whether you are deploying a small application or managing a large-scale enterprise solution.',
          },
          {
            icon: SettingsIcon,
            title: 'Customizable Workflows',
            description:
              'Tailor your deployment workflows to fit your specific requirements, enabling greater flexibility and control over your CI/CD processes.',
          },
        ].map((item, index) => (
          <Box
            key={item.title}
            sx={{
              pr: 4,
              borderRight: {
                xs: 'none',
                md:
                  (index + 1) % 4 !== 0
                    ? `1px solid rgba(255, 255, 255, 0.15)`
                    : 'none',
              },
            }}
          >
            <Box sx={{ display: 'flex', alignItems: 'center', mb: 2 }}>
              <item.icon
                sx={{
                  fontSize: 24,
                  color: 'white',
                  mr: 1,
                }}
              />
              <Typography variant="h3" sx={{ fontSize: 16, fontWeight: 600 }}>
                {item.title}
              </Typography>
            </Box>
            <Typography
              variant="h4"
              sx={{ fontSize: 14 }}
              color="text.secondary"
            >
              {item.description}
            </Typography>
          </Box>
        ))}
      </StyledBox>
    </Box>
  )
}
