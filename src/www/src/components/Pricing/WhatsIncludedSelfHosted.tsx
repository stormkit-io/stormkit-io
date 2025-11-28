import React from 'react'
import CloseIcon from '@mui/icons-material/Close'
import CheckIcon from '@mui/icons-material/Check'

type shTierFn = (t?: number, e?: 'premium' | '') => React.ReactNode

const render = (v: string, enabled: boolean) => (
  <>
    {enabled ? (
      <CheckIcon color="success" sx={{ mr: 2, ml: 0 }} />
    ) : (
      <CloseIcon color="error" sx={{ mr: 2, ml: 0 }} />
    )}
    {v}
  </>
)

const features: shTierFn[] = [
  () => render('Unlimited teams', true),
  () => render('Monitoring', true),
  (_, e) => render('Custom TLS certificates', e === 'premium'),
  () => render('Unlimited domains', true),
  () => render('All other features', true),

  (_, e) => render('Audit logs', e === 'premium'),
  () => render('Unlimited projects', true),
  (_, e) =>
    render(e === 'premium' ? 'Premium support' : 'Community support', true),
  (_, e) => render('Custom features', e === 'premium'),
]

export default features
