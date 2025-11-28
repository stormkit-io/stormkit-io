import React from 'react'
import ReactDOM from 'react-dom/client'
import { BrowserRouter, RouteProps, useLocation } from 'react-router-dom'
import createRoutes from './routes'
import App from './App'
import Context from './context'
import createEmotionCache from './emotion-cache'
import './index.css'

declare global {
  interface Window {
    CONTEXT: any
  }
}

const mount = (children: React.ReactNode) => {
  const root = document.getElementById('root') as HTMLElement

  if (window.CONTEXT) {
    ReactDOM.hydrateRoot(root, children)
  } else {
    ReactDOM.createRoot(root).render(children)
  }
}

const cache = createEmotionCache()

const MyApp = ({ routes }: { routes: RouteProps[] }) => {
  const context = window.CONTEXT
  const location = useLocation()
  const isHomePage = location.pathname === '/'

  return (
    <Context.Provider value={{ ...context, isHomePage }}>
      <App routes={routes} cache={cache} />
    </Context.Provider>
  )
}

;(async () => {
  const { routes } = await createRoutes(window.location.pathname)

  mount(
    <React.StrictMode>
      <BrowserRouter>
        <MyApp routes={routes} />
      </BrowserRouter>
    </React.StrictMode>
  )
})()
