import Box from '@mui/material/Box'
import Divider from '@mui/material/Divider'
import stormkitLogo from '~/assets/logos/stormkit-logo-circle.svg'

interface Props {
  height?: number
}

export default function StormkitConnection({ height = 150 }: Props) {
  return (
    <Box sx={{ maxWidth: '100%', height }}>
      <Divider
        orientation="vertical"
        sx={{ maxWidth: '100%', overflow: 'hidden' }}
      >
        <Box
          component="img"
          sx={{
            width: height / 2.5,
            animation: 'rotate 2.5s ease-in-out infinite',
          }}
          alt="Stormkit Deploy"
          src={stormkitLogo}
        />
      </Divider>
    </Box>
  )
}
