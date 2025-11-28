import Typography from '@mui/material/Typography'
import Box from '@mui/material/Box'
import Header from '~/components/Header'
import Footer from '~/components/Footer'

export const fetchData: FetchDataFunc = async () => {
  return {
    head: {
      title: 'A message from the founder',
      description:
        'Learn how our mission to simplify deployment and vision for flexibility empowers developers, from solo coders to enterprises.',
    },
    context: {},
  }
}

export default function AboutUs() {
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
          sx={{
            fontWeight: 600,
            fontSize: { xs: 24, md: 36 },
            mb: 4,
            textAlign: 'center',
          }}
        >
          A Message from the Founder
        </Typography>
        <Typography sx={{ lineHeight: 1.75 }}>
          Hey there, I'm Savas Vedova, the founder of Stormkit. While I've been
          working at GitLab since 2020, Stormkit is my own separate venture —
          completely unrelated but driven by my passion for creating tools that
          make developers' lives easier. It's a personal project that's close to
          my heart, and I'm excited to see it grow.
        </Typography>
        <Typography sx={{ lineHeight: 1.75, mt: 2 }}>
          At Stormkit, our mission is to simplify the job for developers,
          stripping away the complexity of deployment so you can focus on
          building your product. Our vision is to give you unmatched flexibility
          and control, letting you deploy JavaScript apps — whether solo
          projects or enterprise-scale — on your own terms. I work closely with
          my users to understand their struggles and craft solutions that truly
          fit their needs. Seeing the satisfaction on their faces when Stormkit
          saves them time and effort is actually what fuels my motivation.
        </Typography>
        <Typography sx={{ lineHeight: 1.75, mt: 2 }}>
          Whether you're a solo developer fine-tuning a passion project or part
          of a big enterprise team scaling complex apps, Stormkit is here to
          make deployment a breeze. Give Stormkit a try and feel free to reach
          out to me — I'd love to hear what you think!
        </Typography>
      </Box>
      <Footer maxWidth="lg" />
    </Box>
  )
}
