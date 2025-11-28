import Box, { BoxProps } from '@mui/material/Box'

export default function StyledBox({ children, sx }: BoxProps) {
  return (
    <Box
      sx={{
        mt: 6,
        p: 4,
        background: 'linear-gradient(135deg, #181826 0%, #0c0c1f 100%)',
        bgcolor: '#181826',
        border: '1px solid rgba(255, 255, 255, 0.1)',
        borderRight: 'none',
        borderBottom: 'none',
        borderRadius: 2,
        boxShadow: '0 10px 30px rgba(0, 0, 0, 0.3)',
        ...sx,
      }}
    >
      {children}
    </Box>
  )
}
