import { useEffect, useState } from 'react'
import Box, { BoxProps } from '@mui/material/Box'

interface Props extends Omit<BoxProps, 'position'> {
  children: React.ReactNode
  mode?: 'dark' | 'light'
  position?: 'left' | 'center' | 'right'
  arrowLeftPos?: string
  arrowTopPos?: string
  open?: boolean
}

export default function Tooltip({
  children,
  open = true,
  mode = 'dark',
  arrowLeftPos = '100%',
  arrowTopPos = '50%',
  position,
  sx = {},
  ...rest
}: Props) {
  const isDark = mode === 'dark'
  // @ts-ignore
  let { backgroundColor, color, left, top } = sx
  let defaultLeft = '50%'
  let defaultTop = '100%'
  let defaultTransform = 'translateX(-50%)'
  let defaultMt = 2
  let defaultMl = -1

  if (position === 'left') {
    defaultLeft = '-20%'
    arrowLeftPos = '10%'
    arrowTopPos = '0%'
    defaultTransform = ''
  } else if (position === 'center') {
    defaultLeft = '50%'
    arrowTopPos = '0%'
    arrowLeftPos = '50%'
  }

  if (left || top) {
    defaultTransform = ''
    defaultMl = 0
    defaultMt = 0
  }

  return (
    <Box
      {...rest}
      sx={{
        visibility: open ? 'visible' : 'hidden',
        opacity: open ? 1 : 0,
        position: 'absolute',
        backgroundColor: backgroundColor || (isDark ? 'white' : 'black'),
        color: color || (isDark ? 'black' : 'white'),
        border: '1px solid black',
        borderRadius: 2,
        boxShadow: 20,
        p: 2,
        transition: 'all 0.3s ease',
        maxWidth: 400,
        transform: defaultTransform,
        mt: defaultMt,
        ml: defaultMl,
        left: defaultLeft,
        top: defaultTop,
        ...sx,
        ':after': {
          content: "''",
          position: 'absolute',
          left: arrowLeftPos,
          top: arrowTopPos,
          display: 'block',
          borderColor: backgroundColor || (isDark ? 'white' : 'black'),
          backgroundColor: backgroundColor || (isDark ? 'white' : 'black'),
          transition: 'all 0.3s ease',
          width: '14px',
          height: '14px',
          transform: 'translate(-50%, -50%) rotate(-45deg)',
        },
      }}
    >
      {children}
    </Box>
  )
}
