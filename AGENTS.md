# squire Agent Guide

## Project Overview

- Squire normalizes the project instruction files used by coding agents. It gives a project consistent markerless `AGENTS.md` and `CLAUDE.md` files, keeps them analyzable by stable headings, and stores reusable agent/tool preferences under `~/.config/squire`.

## Technology Stack

- Go 1.25.0 module `squire`
- Go project.

## Project Structure

- `README.md` contains overview.
- `cmd/` contains CLI entrypoints.
- `go.mod` contains Go module.
- `go.sum` contains Go lockfile.
- `internal/` contains private Go packages.
- `squire.yaml` contains Squire project config.

## Commands

- `go test ./...`: Go tests
- `go fmt ./...`: Go format
- `go build ./...`: Go build

## CLI Tools for Agents

- `browser` (`browser`): Browser automation; Use for rendered UI checks/screenshots; examples: `browser open http://localhost:5173`, `browser screenshot`

## Environment and Configuration

- Do not commit secrets unless encrypted workflow exists.
- Prefer documented project commands.
- Start smallest service environment needed.

## Development Workflow

- Read relevant code/docs first.
- Keep scope to requested behavior.
- Update docs when behavior, commands, or structure change.
- Do not overwrite unrelated user changes.

## Coding Standards

- Match style in touched files.
- Add abstractions only for real duplication or complexity.
- Use structured parsers when practical.
- Go code follows existing package boundaries.

## Testing and Verification

- Run narrow checks first; broaden for shared behavior.
- Record reason when checks cannot run.
- Update tests for user-visible or shared behavior.
- Run focused Go tests for touched packages.

## Agent Operating Rules

- Treat this file as project map, not encyclopedia.
- Link source/docs for deep detail.
- Surface missing or conflicting instructions.
- Keep generated section headings stable for `squire analyze` and regeneration.

## Squire Custom Notes

- Add project-specific notes here.
- Squire preserves this section across regeneration.
