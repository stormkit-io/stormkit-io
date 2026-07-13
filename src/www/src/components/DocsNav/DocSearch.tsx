import { useEffect, useState } from 'react'
import TextField from '@mui/material/TextField'
import Box from '@mui/material/Box'
import Chip from '@mui/material/Chip'
import { grey } from '@mui/material/colors'
import SearchIcon from '@mui/icons-material/Search'
import KeyboardCommandKeyIcon from '@mui/icons-material/KeyboardCommandKey'
import DocSearchModal from './DocSearchModal'

export default function DocSearch() {
  const [focused, setFocused] = useState(false)

  useEffect(() => {
    const listener = function (e: any) {
      // Check if CMD (Mac) or CTRL (Windows) key is pressed
      const isCmdOrCtrlPressed = e.metaKey || e.ctrlKey
      const isKKeyPressed = e.key === 'k'

      if (isCmdOrCtrlPressed && isKKeyPressed) {
        setFocused(true)
      }
    }

    document.addEventListener('keydown', listener)

    return () => {
      document.removeEventListener('keydown', listener)
    }
  }, [setFocused])

  return (
    <Box
      sx={{
        flex: 1,
        display: 'flex',
        justifyContent: 'flex-start',
        position: 'relative',
        width: { xs: '100%', md: 'auto' },
      }}
    >
      <Box
        sx={{
          p: { xs: 2, md: 0 },
          borderBottom: { xs: `1px solid ${grey[900]}`, md: 'none' },
          width: { xs: '100%', md: 'auto' },
        }}
      >
        <TextField
          placeholder="Search docs"
          autoComplete="off"
          variant="outlined"
          size="small"
          onClick={() => setFocused(true)}
          sx={{
            userSelect: 'none',
            minWidth: '300px',
            width: { xs: '100%', md: 'auto' },
          }}
          InputProps={{
            startAdornment: <SearchIcon sx={{ mx: 1, fontSize: 14 }} />,
            endAdornment: (
              <Chip
                size="small"
                label={
                  <Box
                    component="span"
                    sx={{ display: 'flex', alignItems: 'center' }}
                  >
                    <KeyboardCommandKeyIcon sx={{ fontSize: 12 }} /> + K
                  </Box>
                }
              />
            ),
            sx: {
              fontSize: 14,
            },
          }}
        />
      </Box>
      {focused && (
        <DocSearchModal
          onClose={() => {
            setFocused(false)
          }}
        />
      )}
    </Box>
  )
}
