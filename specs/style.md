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
  `--gb-aqua` outline, preceded by a squared-plus icon (⊞) indicating a grouping, with its count as an inline pill at the chip's end: `--gb-aqua` fill with `--gb-bg0` text (the inverse of
  the chip), so the count reads as part of the chip rather than a notification badge.
  All task groups share this one color.
- Task group names in the Allocations table use the same chip, without a count.
- Sections on the Job Status page are separated by a faint horizontal rule (`--gb-bg2`) above each section
  heading after the first.
- Status counts in the Versions section are shown as an inline pill at the end of the status badge, inverted
  from the badge: `--gb-bg0` fill with text in the badge's status color.
- The deployment status badge's "deploying" icon is an animated circular-arrow spinner (GIF, since
  it has no live `currentColor` equivalent), colored per theme to match the badge text color
  (`frontend/public/icons/deploying-spinner-{light,dark}.gif`).
- The "deploying" badge doubles as a progress bar: a hard-stop left-to-right gradient driven by the
  `--progress` custom property, `--gb-yellow` for the completed area and `--gb-yellow-dim` (a darker
  yellow defined per theme) for the remainder.
