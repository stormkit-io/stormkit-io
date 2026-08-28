import { useEffect, useState } from 'react'
import Box from '@mui/material/Box'
import Typography from '@mui/material/Typography'

interface Heading {
  id: string
  text: string
}

const slugify = (text: string): string =>
  text
    .toLowerCase()
    .replace(/[^\w\s-]/g, '')
    .trim()
    .replace(/\s+/g, '-')
    .slice(0, 60)

/**
 * Sidebar index of the article's sections, with the one currently on screen
 * highlighted.
 *
 * Headings are read from the rendered DOM rather than the markdown source:
 * the content arrives as an HTML string, and walking it here means the markdown
 * pipeline stays untouched and every existing post gets an index for free.
 */
export default function ArticleToc({ content }: { content?: unknown }) {
  const [headings, setHeadings] = useState<Heading[]>([])
  const [activeId, setActiveId] = useState<string>('')

  useEffect(() => {
    const nodes = Array.from(
      document.querySelectorAll<HTMLHeadingElement>('#blog-content h2')
    )

    const found = nodes.map((node) => {
      if (!node.id) {
        node.id = slugify(node.textContent || '')
      }

      return { id: node.id, text: node.textContent || '' }
    })

    setHeadings(found)

    if (!found.length) {
      return
    }

    // Active section is the last heading to have crossed the top of the
    // viewport, not whichever is most visible. An IntersectionObserver leaves
    // the index blank while the reader is mid-section — every heading is out
    // of frame — whereas this always resolves to exactly one entry.
    const onScroll = () => {
      let current = nodes[0]

      for (const node of nodes) {
        if (node.getBoundingClientRect().top <= 120) {
          current = node
        }
      }

      setActiveId(current.id)
    }

    onScroll()
    window.addEventListener('scroll', onScroll, { passive: true })
    window.addEventListener('resize', onScroll)

    return () => {
      window.removeEventListener('scroll', onScroll)
      window.removeEventListener('resize', onScroll)
    }
  }, [content])

  if (headings.length < 3) {
    return null
  }

  return (
    <Box
      component="nav"
      aria-label="Article sections"
      sx={{
        display: { xs: 'none', lg: 'block' },
        position: 'sticky',
        top: 96,
        alignSelf: 'flex-start',
        width: 232,
        flexShrink: 0,
        pr: 3,
        maxHeight: 'calc(100vh - 140px)',
        overflowY: 'auto',
      }}
    >
      <Typography
        sx={{
          fontSize: 11,
          fontWeight: 600,
          letterSpacing: '0.12em',
          textTransform: 'uppercase',
          opacity: 0.4,
          mb: 1.5,
        }}
      >
        Contents
      </Typography>
      {headings.map(({ id, text }) => (
        <Box
          key={id}
          component="a"
          href={`#${id}`}
          sx={{
            display: 'block',
            fontSize: 13,
            lineHeight: 1.5,
            py: 0.75,
            pl: 1.5,
            borderLeft: '2px solid',
            borderColor: activeId === id ? '#e2ae5a' : 'rgba(255,255,255,0.08)',
            color: activeId === id ? '#e2ae5a' : 'rgba(255,255,255,0.5)',
            textDecoration: 'none',
            transition: 'color 150ms, border-color 150ms',
            ':hover': { color: 'rgba(255,255,255,0.9)' },
          }}
        >
          {text}
        </Box>
      ))}
    </Box>
  )
}
