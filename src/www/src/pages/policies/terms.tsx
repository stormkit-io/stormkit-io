import Typography from '@mui/material/Typography'
import Box from '@mui/material/Box'
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
          Terms of Service
        </Typography>
        <Typography sx={{ opacity: 0.7, mb: 4 }}>
          Effective date: 2023-05-20
        </Typography>
        <Typography>
          These Terms of Service ("Agreement") govern your use of the Stormkit
          deployment and hosting service ("Service") provided by Stormkit OÜ
          ("Company"). By using the Service, you agree to comply with and be
          bound by this Agreement.
        </Typography>{' '}
        <Subtitle>Service Usage</Subtitle>
        <Typography component="div">
          <Typography>
            1.1 You may use the Service to deploy and host your applications on
            our servers.
          </Typography>
          <Typography>
            1.2 You are responsible for the content and activities associated
            with your applications.
          </Typography>
          <Typography>
            1.3 Any violation of this Agreement may result in termination of
            your account and removal of your applications.
          </Typography>
        </Typography>
        <Subtitle>Compliance with Laws and Regulations</Subtitle>
        <Typography component="div">
          <Typography>
            2.1 You must comply with all applicable laws, regulations, and
            third-party rights.
          </Typography>
          <Typography>
            2.2 You agree not to use the Service for any malicious, illegal, or
            harmful activities.
          </Typography>
          <Typography>
            2.3 We reserve the right to take down any reported website that we
            believe to be malicious, violating the law or containing
            explicit/adult content.
          </Typography>
        </Typography>
        <Subtitle>Account Deactivation</Subtitle>
        <Typography component="div">
          <Typography>
            3.1 We reserve the right to deactivate user accounts immediately in
            case of suspected violation of this Agreement or any abusive or
            fraudulent activity.
          </Typography>
          <Typography>
            3.2 Deactivated accounts may result in the loss of data and access
            to the Service.
          </Typography>
        </Typography>
        <Subtitle>Intellectual Property</Subtitle>
        <Typography component="div">
          <Typography>
            4.1 You retain all ownership rights to the content and applications
            you deploy and host using the Service.
          </Typography>
          <Typography>
            4.2 We do not claim ownership over your intellectual property,
            except as necessary to provide the Service.
          </Typography>
        </Typography>{' '}
        <Subtitle>Fees</Subtitle>
        <Typography component="div">
          <Typography>
            5.1 To continue using the Service, you are required to pay the
            applicable fees as determined by the Company.
          </Typography>
          <Typography>
            5.2 Failure to pay the fees may result in suspension or termination
            of your account.
          </Typography>
          <Typography>
            5.3 All fees are non-refundable unless explicitly stated otherwise
            or required by applicable law.
          </Typography>
        </Typography>{' '}
        <Subtitle>Service Availability</Subtitle>
        <Typography component="div">
          <Typography>
            6.1 While we strive to maintain high availability, we do not
            guarantee uninterrupted access to the Service.
          </Typography>
          <Typography>
            6.2 Scheduled maintenance and unforeseen outages may temporarily
            affect service availability.
          </Typography>
        </Typography>
        <Subtitle>Limitation of Liability</Subtitle>
        <Typography component="div">
          <Typography>
            7.1 The Service is provided on an "as is" and "as available" basis.
          </Typography>
          <Typography>
            7.2 We shall not be liable for any direct, indirect, incidental,
            consequential, or exemplary damages arising out of the use or
            inability to use the Service.
          </Typography>
        </Typography>
        <Subtitle>Data Retention</Subtitle>
        <Typography component="div">
          <Typography>
            8.1 Upon termination of your account, we may retain your data for a
            reasonable period to allow for account recovery.
          </Typography>
          <Typography>
            8.2 After this period, your data may be permanently deleted from our
            systems.
          </Typography>
        </Typography>
        <Subtitle>Termination</Subtitle>
        <Typography component="div">
          <Typography>
            9.1 Either party may terminate this Agreement at any time for any
            reason by providing notice to the other party.
          </Typography>
          <Typography>
            9.2 Upon termination, you must cease all use of the Service and
            remove your applications from our servers.
          </Typography>
        </Typography>
        <Subtitle>Governing Law and Dispute Resolution</Subtitle>
        <Typography component="div">
          <Typography>
            10.1 This Agreement shall be governed by and construed in accordance
            with the laws of Estonia.
          </Typography>
          <Typography>
            10.2 Any disputes arising from this Agreement shall be resolved
            through binding arbitration or in the courts of Estonia.
          </Typography>
        </Typography>
        <Subtitle>Contact Information</Subtitle>
        <Typography component="div">
          <Typography>
            11.1 For questions regarding these Terms of Service, please contact
            us at hello@stormkit.io.
          </Typography>
        </Typography>
        <Subtitle>Changes to the Agreement</Subtitle>
        <Typography component="div">
          <Typography>
            12.1 We reserve the right to modify or update this Agreement at any
            time.
          </Typography>
          <Typography>
            12.2 You are responsible for reviewing the most current version of
            the Agreement.{' '}
          </Typography>
          <Typography>
            12.3 Continued use of the Service after any modifications to the
            Agreement constitutes acceptance of the updated terms.
          </Typography>
        </Typography>
        <Typography sx={{ my: 4 }}>
          By using the Service, you acknowledge that you have read, understood,
          and agree to be bound by this Agreement. If you do not agree with any
          part of this Agreement, you may not use the Service.
        </Typography>
      </Box>
      <Footer maxWidth="lg" />
    </Box>
  )
}
