import { useState } from 'react'
import Box from '@mui/material/Box'
import Typography from '@mui/material/Typography'
import Button from '@mui/material/Button'
import { grey } from '@mui/material/colors'
import IconButton from '@mui/material/IconButton'
import ToggleButton from '@mui/material/ToggleButton'
import ToggleButtonGroup from '@mui/material/ToggleButtonGroup'
import ArrowForwardIcon from '@mui/icons-material/ArrowForward'
import ContentCopy from '@mui/icons-material/ContentCopy'
import CheckIcon from '@mui/icons-material/Check'

const MAX_WIDTH_MD = 800

type InstallMode = 'human' | 'agent'

const INSTALL_COMMANDS: Record<InstallMode, string> = {
  human: 'curl -sSL https://www.stormkit.io/install.sh | sh',
  agent: 'curl -sSL https://www.stormkit.io/install.sh | sh -s -- --agent',
}

const MODE_CAPTIONS: Record<InstallMode, string> = {
  human: 'Interactive install. You pick the domain and settings as you go.',
  agent:
    'Hands-off install. Hand your agent an SSH key and it provisions an owner admin plus an MCP-ready API key, no server access needed.',
}

export default function Hero() {
  const [copied, setCopied] = useState(false)
  const [mode, setMode] = useState<InstallMode>('human')

  const command = INSTALL_COMMANDS[mode]

  return (
    <>
      <Typography
        variant="h1"
        sx={{
          fontWeight: 600,
          fontSize: { xs: 24, md: 48 },
          maxWidth: MAX_WIDTH_MD,
          textAlign: 'center',
        }}
      >
        Deploy, Scale, and Own Your Web Apps. No Vendor Lock-In.
      </Typography>
      <Typography
        variant="h2"
        sx={{
          mt: 2,
          fontSize: { xs: 15, md: 17 },
          maxWidth: 700,
          lineHeight: 1.75,
          textAlign: 'center',
          color: grey[400],
        }}
      >
        Stormkit gives development teams full control, faster CI/CD, and up to
        47% lower infrastructure costs, all on your own terms. From
        side-projects, to Enterprise scale.
      </Typography>
      <ToggleButtonGroup
        exclusive
        value={mode}
        size="small"
        onChange={(_, next: InstallMode | null) => {
          if (next) {
            setMode(next)
            setCopied(false)
          }
        }}
        sx={{ mt: 4 }}
      >
        <ToggleButton
          value="human"
          data-umami-event="Install mode: humans"
          sx={{ textTransform: 'none', px: 3 }}
        >
          For Humans
        </ToggleButton>
        <ToggleButton
          value="agent"
          data-umami-event="Install mode: agents"
          sx={{ textTransform: 'none', px: 3 }}
        >
          For Agents
        </ToggleButton>
      </ToggleButtonGroup>
      <Box
        sx={{
          mt: 2,
          fontFamily: 'monospace',
          fontSize: { xs: 10, md: 12 },
          background: 'rgba(255, 255, 255, 0.01)',
          border: '1px solid rgba(255, 255, 255, 0.3)',
          p: 1,
          borderRadius: 2,
          display: 'flex',
          alignItems: 'center',
        }}
      >
        {command}
        <IconButton
          sx={{ ml: 1 }}
          data-umami-event="Self-hosted install script"
          onClick={(e) => {
            e.preventDefault()

            navigator.clipboard.writeText(command)

            setCopied(true)
          }}
        >
          {copied ? (
            <CheckIcon sx={{ fontSize: 14, color: 'green' }} />
          ) : (
            <ContentCopy sx={{ fontSize: 14 }} />
          )}
        </IconButton>
      </Box>
      <Typography
        sx={{
          mt: 1.5,
          fontSize: { xs: 12, md: 13 },
          maxWidth: 560,
          textAlign: 'center',
          color: grey[500],
          minHeight: 40,
        }}
      >
        {MODE_CAPTIONS[mode]}
      </Typography>
      <Box
        sx={{
          maxWidth: MAX_WIDTH_MD,
          textAlign: 'center',
          mt: { xs: 8, md: 4 },
        }}
      >
        <Button
          variant="text"
          color="primary"
          size="large"
          href="https://app.stormkit.io"
          data-umami-event="Cloud login"
          sx={{
            display: { xs: 'none', md: 'inline-flex' },
            mr: { xs: 0, md: 2 },
          }}
        >
          Get started in Cloud
          <ArrowForwardIcon sx={{ mr: 0, ml: 1, fontSize: 16 }} />
        </Button>

        <Button
          variant="contained"
          color="secondary"
          size="large"
          href="/docs/self-hosting/getting-started"
          data-umami-event="Try self-hosted"
        >
          Try Self-Hosted
          <ArrowForwardIcon
            sx={{ mr: 0, ml: 1, fontSize: 16, transform: 'rotate(-45deg)' }}
          />
        </Button>
      </Box>
    </>
  )
}
