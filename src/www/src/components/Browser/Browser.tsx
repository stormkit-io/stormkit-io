import Box from '@mui/material/Box'
import Typography from '@mui/material/Typography'
import LockIcon from '@mui/icons-material/Lock'

interface Props {
  children: React.ReactNode
  url?: string
  actionSlot?: React.ReactNode
  actionBarBgColor?: string
  pageBgColor?: string
}

export default function Browser({
  children,
  url,
  actionSlot,
  actionBarBgColor = 'rgba(0,200,200,0.1)',
  pageBgColor = 'black',
}: Props) {
  return (
    <Box>
      <Box
        sx={{
          bgcolor: actionBarBgColor,
          borderTopRightRadius: 4,
          borderTopLeftRadius: 4,
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          px: 2,
        }}
      >
        <Box sx={{ display: 'flex' }}>
          {['red', 'yellow', 'green'].map((bgcolor) => (
            <Box
              key={bgcolor}
              sx={{
                bgcolor,
                width: 12,
                height: 12,
                borderRadius: '50%',
                opacity: 0.5,
                mr: 1.5,
              }}
            />
          ))}
        </Box>
        <Box
          sx={{
            flexGrow: 1,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
          }}
        >
          {url && (
            <Box
              sx={{
                py: 0.5,
                px: 2,
                m: 1,
                width: '100%',
                backgroundColor: 'rgba(0, 0, 0, 0.3)',
                display: 'flex',
                alignItems: 'center',
                borderRadius: 3,
              }}
            >
              <LockIcon sx={{ mr: 1, color: 'green', width: 16 }} />
              <Typography component="span" sx={{ fontSize: 12 }}>
                {url}
              </Typography>
            </Box>
          )}
          {actionSlot}
        </Box>
      </Box>
      <Box
        sx={{
          fontFamily: 'monospace',
          bgcolor: pageBgColor,
          borderBottomLeftRadius: 4,
          borderBottomRightRadius: 4,
        }}
      >
        {children}
      </Box>
    </Box>
  )
}
