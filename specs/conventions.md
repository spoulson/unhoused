# Conventions

## Code Style
- Follow "[Effective Go](https://go.dev/doc/effective_go)" as a general rule for code style and usage.
- Avoid combining `if` with variable assignment.  e.g. `if err := func(); err != nil { ... }`.
Prefer separate lines per statement.
- Any code not intended to be publicly exported for use outside of this module should reside under an `internal` package.
