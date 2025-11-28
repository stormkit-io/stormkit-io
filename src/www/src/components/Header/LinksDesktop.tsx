import Box from '@mui/material/Box'
import Typography from '@mui/material/Typography'
import ChevronRightIcon from '@mui/icons-material/ChevronRight'
import { useState } from 'react'
import Button from '~/components/Button'
import Tooltip from '~/components/Tooltip'
import links from './links'

let timeout: NodeJS.Timeout

export default function LinksDesktop() {
  const [hovered, setHovered] = useState<number>()

  return (
    <Box
      sx={{
        display: 'flex',
        flexDirection: { xs: 'column', md: 'row' },
        alignItems: 'center',
        ml: 3,
        width: { xs: '100%', md: 'auto' },
        height: { xs: '100%', md: 'auto' },
      }}
    >
      {links.map((link, index) => (
        <Box key={link.path} sx={{ position: 'relative' }}>
          <Button
            id={link.id}
            color="info"
            variant="text"
            onMouseEnter={() => {
              setHovered(index)
            }}
            onClick={() => {
              setHovered(index)
            }}
            sx={{
              mr: 2,
            }}
            href={link.children ? undefined : link.path}
          >
            <Typography component="span" sx={{ fontSize: 14 }}>
              {link.text}
            </Typography>
            {link.children && (
              <ChevronRightIcon sx={{ transform: 'rotate(90deg)' }} />
            )}
          </Button>
          {link.children && (
            <Tooltip
              position={link.position || 'left'}
              open={index === hovered}
              onMouseLeave={() => {
                clearTimeout(timeout)
                timeout = setTimeout(() => {
                  setHovered(undefined)
                }, 250)
              }}
              sx={{
                minWidth: link.maxWidth || 600,
                mt: 4,
                backgroundColor: '#17172b',
              }}
            >
              <Box
                sx={{
                  display: 'grid',
                  gap: 4,
                  gridTemplateColumns: `repeat(auto-fit, minmax(200px, 1fr))`,
                }}
              >
                {link.children.map((section) => (
                  <Box key={section.title}>
                    <Typography
                      variant="subtitle1"
                      color="textSecondary"
                      sx={{ mb: 2 }}
                    >
                      {section.title}
                    </Typography>
                    {section.links.map((btn) => (
                      <Button
                        key={btn.path}
                        variant="text"
                        color="info"
                        href={btn.path}
                        rel={
                          btn.path.startsWith('http')
                            ? 'noreferrer noopener'
                            : undefined
                        }
                        target={
                          btn.path.startsWith('http') ? '_blank' : undefined
                        }
                        sx={{
                          minHeight: 32,
                          display: 'flex',
                          alignItems: 'center',
                          justifyContent: 'flex-start',
                          color: 'white',
                        }}
                      >
                        {btn.icon}
                        {btn.text}
                      </Button>
                    ))}
                  </Box>
                ))}
              </Box>
            </Tooltip>
          )}
        </Box>
      ))}
    </Box>
  )
}
