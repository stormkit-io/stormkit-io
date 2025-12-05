import Box from '@mui/material/Box'
import Grid from '@mui/material/Grid'
import Typography from '@mui/material/Typography'
import PricingTierCard, { type PricingTier } from './PricingTierCard'

const tiers: PricingTier[] = [
  {
    name: 'Free',
    price: 0,
    features: [
      '300 build minutes',
      '100 GB bandwidth',
      '100 GB storage',
      '500,000 function invocations',
    ],
    buttonColor: 'primary',
    borderColor: 'divider',
    borderWidth: '1px',
  },
  {
    name: 'Premium',
    price: 20,
    features: [
      '1,000 build minutes',
      '1 TB bandwidth',
      '1 TB storage',
      '1.5 million function invocations',
      'Access to all features',
    ],
    isPopular: true,
    buttonColor: 'secondary',
    borderColor: 'secondary.main',
    borderWidth: '2px',
  },
  {
    name: 'Ultimate',
    price: 100,
    features: [
      '5,000 build minutes',
      '5 TB bandwidth',
      '5 TB storage',
      '5 million function invocations',
      'Access to all features',
      'Premium support (Slack, Discord, Teams)',
    ],
    buttonColor: 'primary',
    borderColor: 'divider',
    borderWidth: '1px',
  },
]

export default function PricingCloud() {
  return (
    <Box sx={{ px: 2, py: 4 }}>
      <Grid container spacing={3} justifyContent="center">
        {tiers.map((tier) => (
          <PricingTierCard key={tier.name} tier={tier} />
        ))}
      </Grid>
      <Typography
        variant="body2"
        color="text.secondary"
        sx={{ textAlign: 'center', mt: 3 }}
      >
        All limits are per seat. Purchase multiple seats to multiply your
        limits.
      </Typography>
    </Box>
  )
}
