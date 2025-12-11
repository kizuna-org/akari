# Development Guidelines

## Priority Rules

### Workspace Setup

- **Always** `cd /akari/akari/` before working
- **Always** read `Makefile` first

### Code Quality

- Run `make lint` after every iteration
- Maintain ~100% test coverage
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
