# Conventions

## Code Style

- Follow "[Effective Go](https://go.dev/doc/effective_go)" as a general rule for code style and usage.
- Avoid combining `if` with variable assignment.  e.g. `if err := func(); err != nil { ... }`.
Prefer separate lines per statement.
- Any code not intended to be publicly exported for use outside of this module should reside under an `internal` package.
- Code should be readable to easily follow what it is doing.  And comments should be added to explain
_why_ it's doing it, especially if it's not obvious.

## Makefiles

- If necessary to declare a `.PHONY` target, declare it on the line preceding the target.  It is
unfavorable to declare a single `.PHONY` listing all targets.

## Go Tests

- Use Testify `require` and `assert` from module `github.com/stretchr/testify` for test validations.

