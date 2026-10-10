# Development Guidelines

## Priority Rules

### Workspace Setup

- Work in the repository's `akari/` Go module (the checkout path may vary).
- **Always** read `akari/Makefile` before working.
- Behavioral specifications belong in `docs/`; technical architecture belongs in `design/`.
- Preserve confirmed behavioral decisions; consult the user before changing them.

### Code Quality

- Run `make lint` after every iteration
- Maintain ~100% test coverage
- Coverage measures `internal/` packages; validate the thin process entrypoint with a real startup/shutdown smoke check.
- Follow the existing code patterns in the project for consistency.

### Testing Standards

- Use **table-driven tests** for all unit tests
- Never modify mock files instead run `make generate`
- Verify test coverage before commit

## Code Standards

### Architecture

- Follow clean architecture principles
- Don't include other packages within a package
- Use clear package boundaries

### Naming & Files

- Use **camelCase** for file names

### Build & Commands

- Prefer `make` commands over direct `go` commands
- Delete and recreate files to avoid heredoc syntax in terminals
