import { useMemo, useState } from 'react'
import Box from '@mui/material/Box'
import Typography from '@mui/material/Typography'
import ToggleButton from '@mui/material/ToggleButton'
import ToggleButtonGroup from '@mui/material/ToggleButtonGroup'
import PricingCloud from './PricingCloud'
import PricingSelfHosted from './PricingSelfHosted'

type Mode = 'cloud' | 'self-hosted'
type Edition = 'premium' | ''

export default function Pricing() {
  const [mode, setMode] = useState<Mode>('self-hosted')

  const isCloud = useMemo(() => mode === 'cloud', [mode])

  return (
    <Box sx={{ maxWidth: '100%', overflow: 'hidden' }}>
      <Box sx={{ px: 2 }}>
        <Typography
          variant="h2"
          sx={{
            fontWeight: 600,
            fontSize: { xs: 24, md: 48 },
            textAlign: 'center',
            overflow: 'hidden',
            position: 'relative',
          }}
        >
          Simple, predictable pricing
        </Typography>
      </Box>
      <Box>
        <Box sx={{ textAlign: 'center', py: 4 }}>
          <ToggleButtonGroup
            color="success"
            value={mode}
            exclusive
            onChange={(_, newMode) => newMode && setMode(newMode)}
            aria-label="deployment mode"
          >
            <ToggleButton value="cloud">Cloud</ToggleButton>
            <ToggleButton value="self-hosted">Self-Hosted</ToggleButton>
          </ToggleButtonGroup>
        </Box>

        {isCloud && <PricingCloud />}
        {!isCloud && <PricingSelfHosted />}
      </Box>
    </Box>
  )
}
