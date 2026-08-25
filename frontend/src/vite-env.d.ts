/// <reference types="vite/client" />

interface ImportMetaEnv {
  /** Default theme ("light" or "dark") when no user preference is stored yet. See .env.example. */
  readonly VITE_DEFAULT_THEME?: string
}

interface ImportMeta {
  readonly env: ImportMetaEnv
}
