import { useState, useMemo } from 'react'
import Box from '@mui/material/Box'
import Typography from '@mui/material/Typography'
import Slider from '@mui/material/Slider'
import { grey } from '@mui/material/colors'
import { track } from '~/helpers/event'

interface Props {
  seats?: number
  edition?: 'premium' | ''
  onSeatChange?: (v: number) => void
}

let tracked = false

export default function PricingSlider({ seats = 1, edition }: Props) {
  const [value, setValue] = useState<number>(seats)

  const isFirstValue = value === 1

  // The price of the selected tier
  const price = useMemo(() => {
    if (value === 100) {
      return 0
    }

    const isPremium = edition === 'premium'

    if (value === 1 && !isPremium) {
      return -1
    }

    return isPremium ? value * 100 : value * 20
  }, [value, edition])

  // The price label of the selected tier (e.g. up to 100 or 5 seats)
  const priceLabel = useMemo(() => {
    return value === 1
      ? '1 seat'
      : value === 100
      ? '100+ seats'
      : `${value} seats`
  }, [value])

  const handleChange = (_: Event, newValue: number | number[]) => {
    if (typeof newValue === 'number') {
      setValue(newValue)

      if (!tracked) {
        track('Pricing: Self-Hosted', { value: newValue })
        tracked = true
      }
    }
  }

  return (
    <Box sx={{ width: '100%' }}>
      <Box sx={{ display: 'flex', justifyContent: 'space-between', mb: 2 }}>
        <Box sx={{ textAlign: 'left' }}>
          <Typography
            id="pricing-slider"
            gutterBottom
            sx={{ textTransform: 'uppercase', color: grey[500] }}
          >
            Number of seats
          </Typography>
          <Typography sx={{ fontSize: 24, fontWeight: 'bold' }}>
            {priceLabel}
          </Typography>
        </Box>
        <Box sx={{ textAlign: 'right' }}>
          <Typography
            id="pricing-slider"
            gutterBottom
            sx={{ textTransform: 'uppercase', color: grey[500] }}
          >
            Your price
          </Typography>
          <Typography sx={{ fontSize: 24, fontWeight: 'bold' }}>
            {price === -1
              ? 'Free'
              : price === 0
              ? 'Contact us'
              : `$${price} / mo`}
          </Typography>
        </Box>
      </Box>
      <Slider
        sx={{
          ml: isFirstValue ? '10px' : 0,
          mr: 0,
          width: isFirstValue ? 'calc(100% - 10px)' : '100%',
        }}
        color="secondary"
        value={value}
        min={1}
        step={1}
        max={100}
        getAriaValueText={(v: number) => {
          return `${v} seats`
        }}
        valueLabelDisplay="off"
        onChange={handleChange}
        aria-labelledby="pricing-slider"
      />
    </Box>
  )
}
