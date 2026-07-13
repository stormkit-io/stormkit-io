import type { NavigationItem } from './components/DocsNav/DocsNav'
import { createContext } from 'react'

interface Context {
  content?: string
  navigation?: NavigationItem[]
  isHomePage?: boolean
  url?: string
}

export default createContext<Context>({})
