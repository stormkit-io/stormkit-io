import { useEffect } from 'react'
import { useLocation } from 'react-router-dom'

export const useScrollToHash = () => {
  const location = useLocation()

  useEffect(() => {
    const hash = location.hash

    if (hash && hash !== '#') {
      const el = document.querySelector(hash)
      el?.scrollIntoView({ behavior: 'smooth' })
    }
  }, [location.hash])
}
