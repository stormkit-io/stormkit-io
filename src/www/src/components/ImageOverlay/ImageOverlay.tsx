import { useEffect, useState } from 'react'
import Box from '@mui/material/Box'
import Dialog from '@mui/material/Dialog'
import DialogTitle from '@mui/material/DialogTitle'
import IconButton from '@mui/material/IconButton'
import CloseIcon from '@mui/icons-material/Close'
import { NavigationItem } from '../DocsNav/DocsNav'

interface Props {
  content?: { __html: string }
  navigation?: NavigationItem[]
}

export default function ImageOverlay({ content, navigation }: Props) {
  const [hovered, setHovered] = useState<string>()

  useEffect(() => {
    const listener: (this: HTMLElement, ev: MouseEvent) => any = (e) => {
      const el = e.target as HTMLElement
      const wrapper = el.closest('.img-wrapper')
      wrapper && setHovered(wrapper.querySelector('img')?.src)
    }

    document.body.addEventListener('click', listener)

    return () => {
      document.body.removeEventListener('click', listener)
    }
  }, [content, navigation])

  return (
    <Dialog
      fullScreen
      open={Boolean(hovered)}
      sx={{ p: 4 }}
      onClose={() => {
        setHovered(undefined)
      }}
    >
      <DialogTitle
        sx={{
          position: 'absolute',
          right: 0,
          textAlign: 'right',
          bgcolor: 'transparent',
        }}
      >
        <IconButton
          onClick={() => {
            setHovered(undefined)
          }}
        >
          <CloseIcon />
        </IconButton>
      </DialogTitle>
      <Box
        sx={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          width: '100%',
          height: '100%',
          overflowY: 'auto',
          bgcolor: 'background.default',
        }}
      >
        {hovered && (
          <Box
            component="img"
            src={hovered}
            alt="Full screen image"
            sx={{ maxHeight: '100%', maxWidth: '100%' }}
          />
        )}
      </Box>
    </Dialog>
  )
}
