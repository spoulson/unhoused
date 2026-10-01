import { useState } from 'react'

const STORAGE_KEY = 'unhoused-notifications-enabled'

const supported = typeof Notification !== 'undefined'

function initialEnabled(): boolean {
  return supported && localStorage.getItem(STORAGE_KEY) === 'true' && Notification.permission === 'granted'
}

/**
 * The Job Status Page's opt-in deployment-notification toggle: off by default, remembered per-browser
 * (same localStorage-backed pattern as ThemeProvider). Turning it on requests OS notification permission
 * from the click handler — required for the browser to reliably show the permission prompt — and only
 * persists "on" if that's granted.
 */
export function useNotificationsEnabled() {
  const [enabled, setEnabled] = useState<boolean>(initialEnabled)

  function persist(value: boolean) {
    setEnabled(value)
    localStorage.setItem(STORAGE_KEY, String(value))
  }

  async function toggle() {
    if (enabled) {
      persist(false)
      return
    }
    if (!supported) {
      return
    }

    const permission = await Notification.requestPermission()
    persist(permission === 'granted')
  }

  return {
    enabled,
    toggle,
    supported,
    blocked: supported && Notification.permission === 'denied',
  }
}
