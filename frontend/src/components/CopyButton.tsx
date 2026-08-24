import { useState } from 'react'
import styles from './CopyButton.module.css'

interface CopyButtonProps {
  value: string
  label: string
}

export function CopyButton({ value, label }: CopyButtonProps) {
  const [copied, setCopied] = useState(false)

  async function handleClick() {
    try {
      await navigator.clipboard.writeText(value)
      setCopied(true)
      setTimeout(() => setCopied(false), 1500)
    } catch {
      // Clipboard access can fail (permissions, insecure context) — nothing useful to do about it here.
    }
  }

  return (
    <button
      type="button"
      className={`${styles.copyButton} ${copied ? styles.copied : ''}`}
      onClick={handleClick}
      aria-label={label}
      title={copied ? 'Copied!' : label}
    >
      {copied ? '✓' : '⧉'}
    </button>
  )
}
