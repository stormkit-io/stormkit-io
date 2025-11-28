import Button, { ButtonProps } from '@mui/material/Button'
import CircularProgress from '@mui/material/CircularProgress'

interface ButtonEnhancedProps extends ButtonProps {
  loading?: boolean
  target?: '_blank'
}

export default function ButtonEnhanced({
  loading,
  ...rest
}: ButtonEnhancedProps) {
  return (
    <Button {...rest}>
      {loading ? (
        <CircularProgress size="1.25rem" sx={{ color: 'white' }} />
      ) : (
        rest.children
      )}
    </Button>
  )
}
