import type { BoxProps } from '@mui/material'
import React, { useState, useEffect } from 'react'
import Alert from '@mui/material/Alert'
import AlertTitle from '@mui/material/AlertTitle'
import Box from '@mui/material/Box'
import LinearProgress from '@mui/material/LinearProgress'
import ReportIcon from '@mui/icons-material/Report'
import CardHeader from '../CardHeader'
import CardFooter from '../CardFooter'
import CardRow from '../CardRow'
import CardContext, { Size } from './Card.context'

interface Props extends BoxProps {
  successTitle?: string | false
  errorTitle?: string | false
  error?: React.ReactNode
  info?: React.ReactNode
  success?: React.ReactNode
  loading?: boolean
  size?: Size
  onSuccessClose?: () => void
  onErrorClose?: () => void
}

function Card({
  sx,
  errorTitle = 'Error',
  successTitle = 'Success',
  error,
  success,
  info,
  children,
  loading,
  size = 'medium',
  onSuccessClose,
  onErrorClose,
  ...rest
}: Props) {
  let header
  let footer
  let content: React.ReactNode[] = []
  let isCardRow = false
  const [isLoadingFirstTime, setIsLoadingFirstTime] = useState<boolean>()

  useEffect(() => {
    if (typeof isLoadingFirstTime === 'undefined' || isLoadingFirstTime) {
      setIsLoadingFirstTime(loading)
    }
  }, [loading])

  React.Children.forEach(children, (child) => {
    if (!React.isValidElement(child)) {
      return
    }

    if (child.type === CardHeader) {
      header = child
    } else if (child.type === CardFooter) {
      footer = child
    } else {
      // @ts-ignore
      const { children } = child.props || {}

      if (Array.isArray(children) && children[0]?.type === CardRow) {
        isCardRow = true
      }

      content.push(child)
    }
  })

  const p = size === 'small' ? 2 : 4

  return (
    <CardContext.Provider value={{ size }}>
      <Box
        sx={{
          m: 'auto',
          bgcolor: 'page.transparent',
          border: `1px solid`,
          borderColor: 'page.container',
          borderRadius: 2,
          position: 'relative',
          pt: !header && content.length ? p : 0,
          pb: !footer && content.length ? p : 0,
          ...sx,
        }}
        {...rest}
      >
        {header}
        {content && !isLoadingFirstTime && (
          <Box sx={{ px: isCardRow ? 0 : p, flex: 1 }}>{content}</Box>
        )}
        {loading && (
          <Box
            sx={{
              position: 'absolute',
              left: 0,
              right: 0,
              flex: 1,
            }}
          >
            <LinearProgress color="secondary" />
          </Box>
        )}
        {!loading && error && (
          <Alert
            color="error"
            sx={{ mb: footer ? p : 0, px: p, mx: p }}
            onClose={onErrorClose}
          >
            {errorTitle && <AlertTitle>{errorTitle}</AlertTitle>}
            <Box>{error}</Box>
          </Alert>
        )}
        {!loading && success && (
          <Alert
            color="success"
            sx={{ mb: footer ? p : 0, px: p, mx: p }}
            onClose={onSuccessClose}
          >
            {successTitle && <AlertTitle>{successTitle}</AlertTitle>}
            <Box sx={{ flex: 1 }}>{success}</Box>
          </Alert>
        )}
        {!loading && info && (
          <Alert
            color="info"
            icon={<ReportIcon />}
            sx={{
              mb: footer ? p : 0,
              px: p,
              mx: p,
            }}
          >
            <Box>{info}</Box>
          </Alert>
        )}
        {footer && (
          <Box
            sx={{
              px: p,
              py: 2,
              pt: isLoadingFirstTime ? p : 2,
              borderTop: '1px solid',
              borderColor: 'container.border',
            }}
          >
            {footer}
          </Box>
        )}
      </Box>
    </CardContext.Provider>
  )
}

export default Card
