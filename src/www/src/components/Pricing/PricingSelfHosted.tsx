import Box from '@mui/material/Box'
import Grid from '@mui/material/Grid'
import Typography from '@mui/material/Typography'
import PricingTierCard, { type PricingTier } from './PricingTierCard'

const tiers: PricingTier[] = [
  {
    name: 'Free',
    price: 0,
    features: ['Unlimited usage', 'Community support'],
    buttonColor: 'primary',
    borderColor: 'divider',
    borderWidth: '1px',
  },
  {
    name: 'Premium',
    price: 20,
    features: [
      'Unlimited usage',
      'Built-in analytics',
      'Audit logs',
      'Approval mode for user management',
      'Teams',
      'Insights',
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
      'Everything in Premium',
      'Premium support (Slack, Discord, Teams)',
    ],
    buttonColor: 'primary',
    borderColor: 'divider',
    borderWidth: '1px',
  },
]

export default function PricingSelfHosted() {
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
        All features are per seat. Purchase multiple seats for your team.
      </Typography>
    </Box>
  )
}
