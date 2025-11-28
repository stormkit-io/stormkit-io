import React from 'react'
import CheckIcon from '@mui/icons-material/Check'
import { SubscriptionTier } from './SliderCloud'

type cloudTierFn = (t?: SubscriptionTier) => React.ReactNode

const limits: Record<SubscriptionTier, { fns: string; bandwidth: string }> = {
  '100': {
    fns: '2.5m',
    bandwidth: '500GB',
  },
  '500': {
    fns: '5m',
    bandwidth: '1TB',
  },
  '1000': {
    fns: '10m',
    bandwidth: '2TB',
  },
  '1000+': {
    fns: '∞',
    bandwidth: '∞',
  },
}

const render = (v: string) => (
  <>
    <CheckIcon color="success" sx={{ mr: 2, ml: 0 }} />
    {v}
  </>
)

const features: cloudTierFn[] = [
  () => render('Unlimited team members'),
  () => render('Unlimited projects'),
  (tier) => render(`${limits[tier!].bandwidth} Bandwidth`),
  () => render('Unlimited teams'),
  () => render('All features'),
  (tier) => render(`${limits[tier!].fns} serverless invocations`),
  () => render('Unlimited concurrency'),
  () => render('Community support'),
  () => render('30 days deployment retention'),
]

export default features
