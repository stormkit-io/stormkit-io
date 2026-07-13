import type { BoxProps } from '@mui/material'
import React, { useContext } from 'react'
import Box from '@mui/material/Box'
import Chip from '@mui/material/Chip'
import CardContext from '../Card/Card.context'

interface Props extends BoxProps {
  menuLabel?: string
  chipLabel?: React.ReactNode
  icon?: React.ReactNode
  chipColor?: 'success' | 'info' | 'warning'
  actions?: React.ReactNode
}

export default function CardRow({
  sx,
  children,
  menuLabel = 'expand',
  chipLabel,
  chipColor,
  actions,
  icon,
  ...rest
}: Props) {
  const { size } = useContext(CardContext)
  const space = size === 'medium' ? 4 : 2

  return (
    <Box
      sx={{
        borderBottom: '1px solid',
        borderColor: 'container.border',
        display: 'flex',
        alignItems: 'center',
        px: space,
        py: 2,
        ':first-of-type': {
          borderTop: `1px solid`,
          borderColor: 'container.border',
        },
        ':last-child': {
          borderBottom: 'none',
        },
        ...sx,
      }}
      {...rest}
    >
      {icon && <Box sx={{ mr: 2 }}>{icon}</Box>}
      <Box sx={{ flex: 1 }}>{children}</Box>
      {chipLabel && (
        <Chip
          color={chipColor}
          label={chipLabel}
          size="small"
          sx={{ ml: 1, mr: 0 }}
        />
      )}
      {actions && (
        <Box
          sx={{
            display: 'flex',
            alignItems: 'flex-end',
            justifyContent: 'space-between',
            flexDirection: 'column',
          }}
        >
          {actions}
        </Box>
      )}
    </Box>
  )
}
