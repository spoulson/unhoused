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
- Task group counts (per version band and in the Task Groups section) are chips: the group name in a
  `--gb-aqua` outline, preceded by a squared-plus icon (⊞) indicating a grouping, with its count as a larger white-text red (`--gb-red-dim`) bubble badge with a white outline and drop shadow for contrast, on the chip's upper-right corner.
  All task groups share this one color.
- Task group names in the Allocations table use the same chip, without a count bubble.
- Sections on the Job Status page are separated by a faint horizontal rule (`--gb-bg2`) above each section
  heading after the first.
- Status counts in the Versions section are shown as the same red count bubble, on the upper-right corner
  of the status badge.
- The deployment status badge's "deploying" icon is an animated circular-arrow spinner (GIF, since
  it has no live `currentColor` equivalent), colored per theme to match the badge text color
  (`frontend/public/icons/deploying-spinner-{light,dark}.gif`).
- The "deploying" badge doubles as a progress bar: a hard-stop left-to-right gradient driven by the
  `--progress` custom property, `--gb-yellow` for the completed area and `--gb-yellow-dim` (a darker
  yellow defined per theme) for the remainder.
- `--gb-red-dim` is a per-theme red for the count bubbles: same as `--gb-red` in light mode, Gruvbox's
  darker neutral red in dark mode (the bright red has too much contrast against the dark background).
- Count bubbles' text and outline use `--gb-bubble-fg`/`--gb-bubble-ring`: white in light mode, softened
  Gruvbox cream/tan in dark mode.
