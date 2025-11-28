import type { LinkProps } from '@mui/material/Link'
import React from 'react'
import { createTheme } from '@mui/material/styles'
import { Link as RLink, LinkProps as RLinkProps } from 'react-router-dom'
import { green, grey, orange, red } from '@mui/material/colors'

interface PageBackground {
  container: string
  default: string
  paper: string
  transparent: string
}

declare module '@mui/material/styles' {
  interface Palette {
    page: PageBackground
  }

  interface PaletteOptions {
    page: PageBackground
  }
}

const LinkBehavior = React.forwardRef<
  HTMLAnchorElement,
  Omit<RLinkProps, 'to'> & { href: RLinkProps['to'] }
>((props, ref) => {
  const { href, ...other } = props
  // Map href (MUI) -> to (react-router)
  return <RLink ref={ref} to={href} {...other} />
})

export default createTheme({
  typography: {
    allVariants: {
      letterSpacing: 'normal',
      lineHeight: 1.5,
    },
    fontFamily: ['Roboto', 'Helvetica', 'Arial', 'sans-serif'].join(', '),
    subtitle1: {
      fontSize: 12.5,
      textTransform: 'uppercase',
      fontWeight: 600,
    },
    subtitle2: {
      fontSize: 14,
      color: 'white',
    },
  },
  components: {
    MuiLink: {
      defaultProps: {
        component: LinkBehavior,
      } as LinkProps,
      styleOverrides: {
        root: {
          color: orange[400],
          lineHeight: 1.5,
          letterSpacing: 'normal',
        },
      },
    },
    MuiInputLabel: {
      styleOverrides: {
        root: {
          color: 'white',
          '&.Mui-focused': {
            color: 'white',
          },
        },
      },
    },
    MuiOutlinedInput: {
      styleOverrides: {
        root: {
          color: 'white',
          '&.Mui-focused': {
            borderColor: grey[500],
          },
        },
      },
    },
    MuiSvgIcon: {
      styleOverrides: {
        root: {
          color: 'white',
        },
      },
      variants: [
        { props: { color: 'success' }, style: { color: green[600] } },
        { props: { color: 'error' }, style: { color: red[600] } },
      ],
    },
    MuiButton: {
      styleOverrides: {
        root: {
          textTransform: 'none',
          color: grey[100],
          '&:hover': {
            color: grey[100],
          },
          '&.Mui-disabled': {
            color: grey[100],
            opacity: '0.25',
          },
        },
      },
      variants: [
        {
          props: { variant: 'text', color: 'info' },
          style: {
            color: grey[400],
            transition: 'all 0.2s ease-in',
            borderRadius: 10,
            '&:hover': {
              color: grey[100],
              backgroundColor: 'rgba(255,255,255,0.1)',
            },
          },
        },
        {
          props: { variant: 'outlined', color: 'primary' },
          style: {
            color: grey[100],
            border: `1px solid ${grey[100]}`,
            '&:hover': { border: `1px solid ${grey[500]}` },
          },
        },
        {
          props: { variant: 'outlined', color: 'info' },
          style: {
            color: grey[500],
            borderRadius: 6,
            transition: 'all 0.2s ease-in',
            border: `1px solid ${grey[800]}`,
            '&:hover': { backgroundColor: 'rgba(255, 255, 255, 0.1)' },
          },
        },
      ],
    },
  },
  palette: {
    mode: 'dark',
    primary: {
      main: '#0F092B',
      light: '#e7e5ee',
      contrastText: '#e7e5ee',
    },
    secondary: {
      main: '#78193B',
      dark: '#c5245e',
    },
    text: {
      primary: '#a4a4a4',
      secondary: '#999999',
    },
    background: {
      default: 'transparent',
      paper: '#262558',
    },
    info: {
      dark: '#7272f1',
      main: grey[900],
      contrastText: grey[100],
    },
    page: {
      container: 'rgb(11, 6, 30)',
      default: '',
      paper: '',
      transparent: 'rgba(255,255,255,0.05)',
    },
  },
  breakpoints: {
    values: {
      xs: 0,
      sm: 384,
      md: 576,
      lg: 1024,
      xl: 1200,
    },
  },
})
