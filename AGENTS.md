# Development Guidelines

## Development Task Tracking

- Use the Akari MCP task tools (`tasks_list`, `tasks_get`, `tasks_update`) as the source of truth when checking or updating development tasks. Do not infer the current task list from GitHub issues, local files, or the console when MCP is available.
- Before starting implementation from a task, read its details and referenced docs/design through Akari MCP and confirm prerequisites/status. Keep decision or blocked tasks unassigned until their dependency is resolved.

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
