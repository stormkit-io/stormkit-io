import { useState } from 'react'
import Box from '@mui/material/Box'
import ChevronRightIcon from '@mui/icons-material/ChevronRight'
import StyledBox from '~/components/StyledBox'
import Typography from '@mui/material/Typography'
import { track } from '~/helpers/event'

const faq = [
  {
    question: 'What is Stormkit?',
    answer:
      'Stormkit is a hosting and deployment platform for modern web applications, offering seamless integration with Git repositories and powerful CI/CD capabilities.',
  },
  {
    question: 'Which frameworks does Stormkit support?',
    answer:
      'The self-hosted version supports both server-side and client-side frameworks and runtimes. It is highly configurable, so it makes it possible to host a wide range of web applications.',
  },
  {
    question: 'How does deployment work?',
    answer:
      'Simply connect your Git repository to Stormkit. Every push to your configured branch triggers an automatic deployment with zero configuration required.',
  },
  {
    question: 'Do you offer custom domains?',
    answer:
      'Yes, you can connect custom domains to your applications with automatic TLS certificate provisioning.',
  },
  {
    question: 'What about environment variables?',
    answer:
      'Stormkit supports securely configuring the environment variables for client-side and server-side applications.',
  },
  {
    question: 'Is there a free tier?',
    answer:
      'Yes, the self-hosted version comes with a free seat. Credit card not required.',
  },
  {
    question: 'How do I get started?',
    answer:
      'Sign up for a Stormkit account, connect your Git repository, configure your build settings, and deploy your first application in minutes.',
  },
  {
    question: 'What kind of support do you provide?',
    answer:
      'We offer documentation, community support, and email support for all users, with priority support available for paid plans, including direct contact through Slack, Discord or other messaging applications.',
  },
  {
    question: 'What is the difference between Stormkit and other providers?',
    answer:
      'Stormkit stands out from other providers by offering an enterprise-grade self-hostable solution with customizable deployment options and no vendor lock-in.',
  },
]

export default function FAQ() {
  const [openIndex, setOpenIndex] = useState(-1)

  return (
    <Box sx={{ px: { xs: 2, md: 0 } }}>
      <Typography
        variant="h2"
        sx={{
          fontSize: { xs: 16, md: 32 },
          fontWeight: 600,
          textAlign: 'center',
          mb: 4,
        }}
      >
        Frequently asked questions (FAQ)
      </Typography>

      {faq.map((f, index) => (
        <StyledBox
          key={f.question}
          sx={{
            mt: 0,
            mb: 2,
            p: 2,
            cursor: 'pointer',
            '&:hover': {
              backgroundColor: 'rgba(0, 0, 0, 0.04)',
            },
          }}
          onClick={() => {
            setOpenIndex(openIndex === index ? -1 : index)
          }}
        >
          <Box
            tabIndex={0}
            onClick={() => {
              setOpenIndex(openIndex === index ? -1 : index)
              track('FAQ click', { index })
            }}
          >
            <Box sx={{ display: 'flex' }}>
              <Box sx={{ flex: 1 }}>{f.question}</Box>
              <ChevronRightIcon
                sx={{
                  transform:
                    openIndex === index ? 'rotate(-90deg)' : 'rotate(90deg)',
                }}
              />
            </Box>
          </Box>
          <Box
            color="text.secondary"
            sx={{
              mt: 2,
              display: openIndex === index ? 'block' : 'none',
            }}
          >
            {f.answer}
          </Box>
        </StyledBox>
      ))}
    </Box>
  )
}
