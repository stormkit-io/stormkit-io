import { useState } from 'react'
import Box from '@mui/material/Box'
import Typography from '@mui/material/Typography'
import Slider from '@mui/material/Slider'
import { track } from '~/helpers/event'

type AllowedValues = 1 | 101 | 201 | 301

export type SubscriptionTier = '100' | '500' | '1000' | '1000+'

interface Props {
  tier?: SubscriptionTier
  onTierChange?: (v: SubscriptionTier) => void
}

const labelMap: Record<AllowedValues, SubscriptionTier> = {
  '1': '100',
  '101': '500',
  '201': '1000',
  '301': '1000+',
}

const labelMapInverse: Record<SubscriptionTier, AllowedValues> =
  Object.fromEntries(Object.entries(labelMap).map((a) => a.reverse()))

const priceMap: Record<AllowedValues, number> = {
  '1': 20,
  '101': 75,
  '201': 150,
  '301': 0,
}

function valueLabelFormat(value: number) {
  // @ts-ignore
  return labelMap[value]
}

let tracked = false

export default function PricingSlider({ tier = '100', onTierChange }: Props) {
  const [value, setValue] = useState<AllowedValues>(
    (tier ? Number(labelMapInverse[tier]) : 1) as AllowedValues
  )

  const handleChange = (event: Event, newValue: number | number[]) => {
    if (typeof newValue === 'number') {
      setValue(newValue as AllowedValues)
      onTierChange?.(labelMap[newValue as AllowedValues])

      if (!tracked) {
        track('Pricing: Cloud', { value: newValue })
        tracked = true
      }
    }
  }

  const isFirstValue = value === 1
  const isLastValue = value === 301

  return (
    <Box sx={{ width: '100%' }}>
      <Box sx={{ display: 'flex', justifyContent: 'space-between', mb: 2 }}>
        <Box sx={{ textAlign: 'left' }}>
          <Typography
            id="pricing-slider"
            gutterBottom
            sx={{ textTransform: 'uppercase', opacity: 0.5 }}
          >
            Monthly deployments
          </Typography>
          <Typography sx={{ fontSize: 24, fontWeight: 'bold' }}>
            {isLastValue ? 'from' : 'up to'} {labelMap[value]}
          </Typography>
        </Box>
        <Box sx={{ textAlign: 'right' }}>
          <Typography
            id="pricing-slider"
            gutterBottom
            sx={{ textTransform: 'uppercase', opacity: 0.5 }}
          >
            Your price
          </Typography>
          <Typography sx={{ fontSize: 24, fontWeight: 'bold' }}>
            {priceMap[value] ? `$${priceMap[value]} / mo` : 'Contact us'}
          </Typography>
        </Box>
      </Box>
      <Slider
        sx={{
          ml: isFirstValue ? '10px' : 0,
          mr: isLastValue ? '10px' : 0,
          width: isFirstValue || isLastValue ? 'calc(100% - 10px)' : '100%',
        }}
        color="secondary"
        value={value}
        min={1}
        step={100}
        max={301}
        getAriaValueText={valueLabelFormat}
        valueLabelDisplay="off"
        onChange={handleChange}
        aria-labelledby="pricing-slider"
      />
    </Box>
  )
}
