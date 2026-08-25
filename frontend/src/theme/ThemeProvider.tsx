import { useLayoutEffect, useState, type ReactNode } from 'react'
import { ThemeContext, type Theme } from './ThemeContext'

const STORAGE_KEY = 'unhoused-theme'

function isTheme(value: string | null | undefined): value is Theme {
  return value === 'light' || value === 'dark'
}

/** The app's default theme, set via VITE_DEFAULT_THEME (see .env.example). */
function configuredDefaultTheme(): Theme {
  const configured = import.meta.env.VITE_DEFAULT_THEME
  return isTheme(configured) ? configured : 'light'
}

function initialTheme(): Theme {
  const stored = localStorage.getItem(STORAGE_KEY)
  if (isTheme(stored)) {
    return stored
  }
  return configuredDefaultTheme()
}

export function ThemeProvider({ children }: { children: ReactNode }) {
  const [theme, setTheme] = useState<Theme>(initialTheme)

  // useLayoutEffect (not useEffect) so the attribute is set before the browser paints,
  // avoiding a flash of the wrong theme on load.
  useLayoutEffect(() => {
    document.documentElement.setAttribute('data-theme', theme)
    localStorage.setItem(STORAGE_KEY, theme)
  }, [theme])

  function toggleTheme() {
    setTheme((prev) => (prev === 'light' ? 'dark' : 'light'))
  }

  return <ThemeContext.Provider value={{ theme, toggleTheme }}>{children}</ThemeContext.Provider>
}
