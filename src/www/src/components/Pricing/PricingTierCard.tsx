import Box from '@mui/material/Box'
import Grid from '@mui/material/Grid'
import Typography from '@mui/material/Typography'
import Divider from '@mui/material/Divider'
import Button from '@mui/material/Button'
import Chip from '@mui/material/Chip'

export interface PricingTier {
  name: string
  price: number
  features: string[]
  isPopular?: boolean
  buttonColor: 'primary' | 'secondary'
  borderColor: string
  borderWidth: string
}

interface PricingTierCardProps {
  tier: PricingTier
}

export default function PricingTierCard({ tier }: PricingTierCardProps) {
  return (
    <Grid key={tier.name} size={{ xs: 12, md: 4 }}>
      <Box
        sx={{
          border: `${tier.borderWidth} solid`,
          borderColor: tier.borderColor,
          borderRadius: 2,
          p: 3,
          height: '100%',
          display: 'flex',
          flexDirection: 'column',
          position: 'relative',
          transition: 'transform 0.2s ease-in-out',
          '&:hover': {
            transform: 'scale(1.05)',
          },
        }}
      >
        {tier.isPopular && (
          <Chip
            label="Popular"
            color="secondary"
            size="small"
            sx={{ position: 'absolute', top: -12, right: 16 }}
          />
        )}
        <Typography variant="h5" sx={{ fontWeight: 600, mb: 1 }}>
          {tier.name}
        </Typography>
        <Typography variant="h3" sx={{ fontWeight: 700, mb: 3 }}>
          ${tier.price}
          <Typography component="span" variant="body2" color="text.secondary">
            /month
          </Typography>
        </Typography>
        <Divider sx={{ mb: 3 }} />
        <Box sx={{ flex: 1 }}>
          {tier.features.map((feature) => (
            <Typography
              key={feature}
              variant="body2"
              color="text.secondary"
              sx={{ mb: 1 }}
            >
              • {feature}
            </Typography>
          ))}
        </Box>
        <Button
          variant="contained"
          color={tier.buttonColor}
          size="large"
          sx={{ mt: 5 }}
          href="https://app.stormkit.io"
        >
          Get started
        </Button>
      </Box>
    </Grid>
  )
}
