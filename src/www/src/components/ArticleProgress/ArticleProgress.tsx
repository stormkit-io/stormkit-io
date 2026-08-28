import { useEffect, useState } from 'react'
import Box from '@mui/material/Box'

/**
 * A thin bar across the top of the viewport that fills as the reader moves
 * through the article. Long-form posts give no sense of how much is left:
 * the scrollbar is hidden on most systems and the footer is a poor proxy.
 */
export default function ArticleProgress() {
  const [progress, setProgress] = useState(0)

  useEffect(() => {
    const onScroll = () => {
      const scrollable =
        document.documentElement.scrollHeight - window.innerHeight

      setProgress(scrollable > 0 ? (window.scrollY / scrollable) * 100 : 0)
    }

    onScroll()
    window.addEventListener('scroll', onScroll, { passive: true })
    window.addEventListener('resize', onScroll)

    return () => {
      window.removeEventListener('scroll', onScroll)
      window.removeEventListener('resize', onScroll)
    }
  }, [])

  return (
    <Box
      aria-hidden
      sx={{
        position: 'fixed',
        top: 0,
        left: 0,
        right: 0,
        height: '2px',
        zIndex: (theme) => theme.zIndex.appBar + 1,
        pointerEvents: 'none',
      }}
    >
      <Box
        sx={{
          height: '100%',
          width: `${progress}%`,
          background: 'linear-gradient(90deg, #e2ae5a 0%, #ef476f 100%)',
          boxShadow: '0 0 12px rgba(226, 174, 90, 0.6)',
          transition: 'width 80ms linear',
        }}
      />
    </Box>
  )
}
