import Typography from '@mui/material/Typography'
import Box from '@mui/material/Box'
import Link from '@mui/material/Link'
import Header from '~/components/Header'
import Footer from '~/components/Footer'

interface Props {
  children: React.ReactNode
}

function Subtitle({ children }: Props) {
  return (
    <Typography
      variant="subtitle2"
      sx={{ fontSize: 20, fontWeight: 600, mt: 4 }}
    >
      {children}
    </Typography>
  )
}

export default function Terms() {
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
          bgcolor: 'rgba(0,0,0,0.1)',
          my: 4,
          p: { xs: 2, md: 4 },
        }}
        maxWidth="768px"
      >
        <Typography
          variant="h1"
          sx={{ fontWeight: 600, fontSize: { xs: 24, md: 36 }, mb: 4 }}
        >
          Privacy Policy
        </Typography>
        <Typography sx={{ opacity: 0.7, mb: 4 }}>
          Effective date: 2023-05-20
        </Typography>{' '}
        <Typography>
          At Stormkit, we respect the privacy of our users and are committed to
          protecting their personal information. This Privacy Policy outlines
          how we collect, use, disclose, and safeguard the information you
          provide while using our services.
        </Typography>{' '}
        <Subtitle>Information Collection and Use</Subtitle>
        <Typography component="div">
          We may collect personal information, such as your name, email address,
          and contact details, when you voluntarily provide it to us. This
          information is used to communicate with you, fulfill your requests,
          and improve our services. We may also collect non-personal
          information, such as your browser type and IP address, to enhance your
          user experience and analyze usage patterns on our website.
        </Typography>
        <Subtitle>Information Sharing and Disclosure</Subtitle>
        <Typography component="div">
          We do not sell, trade, or rent your personal information to third
          parties. However, we may share your information with trusted service
          providers who assist us in operating our business, conducting surveys
          or contests, or providing customer support. We may also disclose your
          information if required by law or to protect our rights, property, or
          safety.
        </Typography>
        <Subtitle>Data Security</Subtitle>
        <Typography component="div">
          We employ industry-standard security measures to safeguard your
          personal information and protect it from unauthorized access,
          disclosure, alteration, or destruction. However, no method of
          transmission over the internet or electronic storage is 100% secure,
          and we cannot guarantee absolute security.
        </Typography>
        <Subtitle>Cookies</Subtitle>
        <Typography component="div">
          Our website may use cookies and similar technologies to enhance your
          browsing experience and collect information about how you use our
          services. You have the option to disable cookies through your browser
          settings, but this may affect certain features or functionality of our
          website.
        </Typography>{' '}
        <Subtitle>Anonymous User Behavior Tracking</Subtitle>
        <Typography component="div">
          We may collect and analyze anonymous user behavior data to understand
          how users interact with our services, identify usage patterns, and
          improve our platform's performance and user experience. This data is
          aggregated and anonymized, meaning it cannot be used to identify
          individual users. The information collected may include page views,
          click patterns, feature usage, and general navigation behavior within
          our application.
        </Typography>{' '}
        <Subtitle>External Links</Subtitle>
        <Typography component="div">
          Our website may contain links to third-party websites. We have no
          control over the content or practices of these websites and are not
          responsible for their privacy policies. We encourage you to review the
          privacy policies of any third-party websites you visit.
        </Typography>{' '}
        <Subtitle>Children's Privacy</Subtitle>
        <Typography component="div">
          Our services are not intended for individuals under the age of 13. We
          do not knowingly collect personal information from children. If you
          are a parent or guardian and believe your child has provided us with
          personal information, please contact us, and we will take appropriate
          steps to remove the information from our records.
        </Typography>
        <Subtitle>Changes to This Privacy Policy</Subtitle>
        <Typography component="div">
          We reserve the right to update or modify this Privacy Policy at any
          time. Any changes will be effective immediately upon posting the
          revised policy on our website. We encourage you to review this Privacy
          Policy periodically to stay informed about how we are protecting your
          information.
        </Typography>
        <Typography sx={{ mt: 4 }}>
          If you have any questions or concerns about this Privacy Policy,
          please contact us at{' '}
          <Link color="secondary" href="mailto:hello@stormkit.io">
            hello@stormkit.io
          </Link>
          .
        </Typography>
        <Typography sx={{ my: 4 }}>
          By using our services, you consent to the collection, use, and
          disclosure of your information as described in this Privacy Policy.
        </Typography>
      </Box>
      <Footer maxWidth="lg" />
    </Box>
  )
}
