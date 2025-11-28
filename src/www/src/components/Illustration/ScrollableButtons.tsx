import type { FeatureProps } from './features'
import { useEffect } from 'react'
import { useState } from 'react'
import Box from '@mui/material/Box'
import Button from '@mui/material/Button'
import IconButton from '@mui/material/IconButton'
import ArrowBackIosNewIcon from '@mui/icons-material/ArrowBackIosNew'
import ArrowForwardIosIcon from '@mui/icons-material/ArrowForwardIos'

interface Props {
  buttons: FeatureProps[]
  activeButton: number
  setActiveButton: (index: number) => void
}

export default function ScrollableButtons({
  buttons,
  activeButton,
  setActiveButton,
}: Props) {
  const [left, setLeft] = useState(0)

  // Register keydown event listener for navigating with arrows
  useEffect(() => {
    const keydown = (event: KeyboardEvent) => {
      if (event.key === 'ArrowRight') {
        setActiveButton(Math.min(activeButton + 1, buttons.length - 1))
      } else if (event.key === 'ArrowLeft') {
        setActiveButton(Math.max(activeButton - 1, 0))
      }
    }

    document.body.addEventListener('keydown', keydown)

    return () => {
      document.body.removeEventListener('keydown', keydown)
    }
  }, [activeButton, buttons])

  // Scroll to active button
  useEffect(() => {
    const buttons = document.querySelectorAll('.feature-button')
    let activeButtonIndex = activeButton

    while (activeButtonIndex >= 0) {
      const button = buttons[activeButtonIndex] as HTMLButtonElement
      const lastButton = buttons[buttons.length - 1] as HTMLButtonElement

      const parentSize = button.parentElement?.offsetWidth || 0

      if (
        button &&
        parentSize + button?.offsetLeft <=
          lastButton.offsetLeft + lastButton.offsetWidth + 170
      ) {
        setLeft(-button.offsetLeft)
        return
      }

      activeButtonIndex = activeButtonIndex - 1
    }
  }, [activeButton])

  return (
    <Box
      sx={{
        display: 'flex',
        alignItems: 'center',
        position: 'relative',
        p: 4,
        pt: 0,
      }}
    >
      <IconButton
        onClick={() => {
          setActiveButton(0)
        }}
        sx={{
          position: 'absolute',
          left: 0,
          zIndex: 2,
          boxShadow: 1,
          '&:hover': { bgcolor: 'background.paper' },
        }}
      >
        <ArrowBackIosNewIcon fontSize="small" />
      </IconButton>

      {/* Scrollable container */}
      <Box
        sx={{
          whiteSpace: 'nowrap',
          maxWidth: { xs: 300, md: 500, lg: 800 },
          mx: 3, // Space for arrows
          position: 'relative',
          overflowX: 'hidden',
        }}
      >
        <Box
          className="scrollable-buttons"
          sx={{ position: 'relative', left, transition: 'all .3s' }}
        >
          {buttons.map((button, index) => (
            <Button
              id={button.id}
              key={button.id}
              variant="outlined"
              color="primary"
              className="feature-button"
              onClick={() => {
                setActiveButton(index)
              }}
              sx={{
                display: 'inline-block',
                whiteSpace: 'nowrap',
                opacity: activeButton === index ? 1 : 0.5,
                transition: 'opacity 0.3s',
                ml: 2,
                ':first-of-type': { ml: 0 },
              }}
            >
              {button.title}
            </Button>
          ))}
        </Box>
      </Box>
      <IconButton
        onClick={() => {
          if (activeButton === buttons.length - 1) {
            setActiveButton(0)
          } else {
            setActiveButton(activeButton + 1)
          }
        }}
        className="scroll-right"
        sx={{
          position: 'absolute',
          right: 0,
          zIndex: 2,
          boxShadow: 1,
          '&:hover': { bgcolor: 'background.paper' },
        }}
      >
        <ArrowForwardIosIcon fontSize="small" />
      </IconButton>
    </Box>
  )
}
