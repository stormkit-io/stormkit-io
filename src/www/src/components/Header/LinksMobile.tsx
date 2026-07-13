import Box from '@mui/material/Box'
import Typography from '@mui/material/Typography'
import { grey } from '@mui/material/colors'
import GitHubIcon from '@mui/icons-material/GitHub'
import ArrowForwardIcon from '@mui/icons-material/ArrowForward'
import Button from '~/components/Button'
import links from './links'

export default function LinksMobile() {
  return (
    <Box
      sx={{
        display: 'flex',
        flexDirection: 'column',
        alignItems: 'center',
        width: '100%',
      }}
    >
      {links.map((link, index) => (
        <Box
          key={link.path}
          sx={{
            position: 'relative',
            borderBottom: `1px solid ${grey[900]}`,
            width: '100%',
          }}
        >
          {!link.children && (
            <Button
              id={link.id}
              color="info"
              variant="text"
              sx={{
                mr: 2,
                py: 2,
              }}
              href={link.children ? undefined : link.path}
            >
              <Typography
                component="span"
                sx={{
                  fontSize: 11,
                  textTransform: 'uppercase',
                  fontWeight: 'bold',
                  pl: 2,
                }}
              >
                {link.text}
              </Typography>
            </Button>
          )}
          {link.children && (
            <Box
              sx={{
                display: 'grid',
                gap: 2,
                gridTemplateColumns: `repeat(auto-fit, minmax(${
                  link.children?.length * 80
                }px, 1fr))`,
              }}
            >
              {link.children.map((section) => (
                <Box key={section.title} sx={{ px: 3, py: 2 }}>
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
          )}
        </Box>
      ))}
      <Box
        sx={{
          flex: 1,
          fontSize: 12.5,
          display: 'flex',
          alignItems: 'center',
          bgcolor: 'transparent',
          zIndex: 1,
        }}
      >
        <Box
          sx={{
            display: 'flex',
            flex: 1,
            mt: 2,
          }}
        >
          <Button
            variant="outlined"
            color="info"
            href="https://github.com/stormkit-io"
            rel="noreferrer noopener"
            target="_blank"
            size="small"
            sx={{
              display: 'inline-flex',
              mr: 2,
            }}
          >
            <GitHubIcon sx={{ mr: 1, fontSize: 14 }} />
            GitHub
            <ArrowForwardIcon
              sx={{ ml: 1, transform: 'rotate(-45deg)', fontSize: 14 }}
            />
          </Button>
          <Button
            variant="outlined"
            color="info"
            href="/contact"
            sx={{ mr: 2 }}
          >
            Contact
          </Button>
          <Button
            variant="outlined"
            color="info"
            href="https://app.stormkit.io"
            data-umami-event="Cloud login"
          >
            Login
            <ArrowForwardIcon
              sx={{ ml: 1, fontSize: 14, transform: 'rotate(-45deg)' }}
            />
          </Button>
        </Box>
      </Box>
    </Box>
  )
}
