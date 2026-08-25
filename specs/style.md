## Style and Appearance

- Easily readable layout.
- Use color theme "Gruvbox" (See: https://gruvbox.org/), with both its light and dark modes
  selectable — light uses Gruvbox's "faded" accent colors (its own recommendation for light
  backgrounds), dark uses the "bright" accent colors (its recommendation for dark backgrounds).
- A toggle button in the header switches between light/dark mode immediately, without a page
  reload. The chosen mode is remembered per-browser (persisted client-side) and takes precedence
  over the configured default on future visits.
- Frontend configuration (`VITE_DEFAULT_THEME` build-time env var, see `frontend/.env.example`)
  sets which mode is shown to a visitor who hasn't chosen one yet. Defaults to light if unset.
- Use Kefa font for most text.
- Use AnonymicePro Nerd Font for monospaced text.

