# Squire

Squire normalizes the project instruction files used by coding agents. It gives a project consistent markerless `AGENTS.md` and `CLAUDE.md` files, keeps them analyzable by stable headings, and stores reusable agent/tool preferences under `~/.config/squire` (or `~/Library/Application Support/squire` on macOS).

## Install

```bash
go install github.com/AndreBaltazar8/squire/cmd/squire@latest
```

This drops the `squire` binary in `$(go env GOBIN)` (or `$(go env GOPATH)/bin`, default `~/go/bin`). Make sure that directory is on your `PATH`. Squire targets the Go toolchain declared in `go.mod`; older installed Go versions still work as long as `GOTOOLCHAIN=auto` (the default since Go 1.21) is in effect.

To work from a clone instead:

```bash
git clone https://github.com/AndreBaltazar8/squire.git
cd squire
go install ./cmd/squire
```

## Why

Different agents read different files:

- Codex and other tools commonly use `AGENTS.md`.
- Claude Code reads `CLAUDE.md`.
- Cursor Agent is configured by project rules, but can still benefit from a shared project guide.

Squire treats `AGENTS.md` as the shared source of truth and generates `CLAUDE.md` as a thin Claude-specific wrapper that imports it.

It detects root project files and immediate `src/*` subprojects, including Go services and SvelteKit/Vite apps, and writes concise project-structure phrases.
When a project has `DESIGN.md` or `design.md`, Squire surfaces it as the visual design system source of truth so agents read design tokens and rationale before frontend/UI changes.

## Commands

```bash
# See configured agents and CLI tools, with PATH detection
squire detect

# Generate AGENTS.md and CLAUDE.md in the current project
squire generate all

# Select reusable guidance for a new/empty project
squire generate all -i --project-name myapp

# Generate with explicit components
squire generate all --component svelte --component go-api --project-name myapp

# Preview output without writing
squire generate all --stdout

# Download public component definitions
squire download AndreBaltazar8/squire-components
squire download AndreBaltazar8/squire-components#svelte

# Browse public component providers
squire browse
squire browse -i
squire browse AndreBaltazar8/squire-components#svelte

# Manage installed components
squire component list
squire component remove svelte

# Check section coverage in existing files
squire analyze

# Manage CLI tools that generated guides expose to agents
squire cli list
squire cli add playwright --description "Rendered UI helper." --when "Use for rendered UI verification." --example "playwright test"
squire cli remove playwright
```

By default, Squire auto-detects components from current project files. Add a project-root `squire.yaml` when a project should pin components explicitly, and use `--component <id>` for one-off additions during a generation. `generate -i` opens a searchable selector.

## Markerless Guides

Generated `AGENTS.md` and `CLAUDE.md` files are plain Markdown with stable section headings.

`squire analyze` uses headings to detect missing required sections. Regeneration also uses headings to preserve the `Squire Custom Notes` section and to decide whether an existing markerless guide has Squire's expected shape.

Generated `AGENTS.md` includes a `Squire Custom Notes` section. Add project-specific notes there; Squire preserves that section across regeneration.

## Configuration

Squire creates its global config file on first use:

- `~/.config/squire/config.yaml`

Optional components live in `~/.config/squire/components/*.yaml`. Components can detect files or be selected during generation, then add guidance, commands, and CLI tools to generated agent files.

Project-local settings live in `squire.yaml`:

```yaml
components:
  - go
  - svelte
description: |
  myapp is a tiny in-memory key/value store with a JSON HTTP surface
  and a small Go client library. It is used by services in the same
  monorepo for ephemeral coordination state.
```

`description`, when set, replaces the auto-detected Project Overview paragraph. Pin it when the README's first paragraph is not actually a description of the project — for example, when the README opens with a setup section, a "How to get started" prefix, or a meta disclaimer about the README itself.

Use `squire browse` to list public providers from `AndreBaltazar8/squire-components`. Use `squire browse -i` to search and install interactively. Browse reads `index.yaml` when present, then falls back to provider discovery. Use `squire download <owner>/<repo>` to install all components from a GitHub provider, or append `#component-id` to install one. A provider can define `provider.yaml` with `components: <dir>`; otherwise Squire looks for `components/`.

```yaml
version: 1
id: svelte
detectors:
  any:
    - svelte.config.js
    - svelte.config.ts
guidance:
  technologies:
    - Frontend uses SvelteKit.
  design:
    - Read `DESIGN.md` before frontend/UI changes when present.
```

Detector entries are either a bare path/glob (existence check) or an object with `file`, `contains`, or `regex` for content-aware matching. Use the latter to avoid false positives when a generic file name doesn't tell you which technology is in play:

```yaml
detectors:
  any:
    - pg_hba.conf                       # bare glob: file must exist
    - file: docker-compose.yml          # content match: file exists AND
      contains: postgres                # body contains substring (case-insensitive)
    - file: package.json
      regex: '"(pg|postgres)"\s*:'      # or matches regex (also case-insensitive)
```

Add global CLI tools with `squire cli add` or by editing `config.yaml`:

```yaml
cli_tools:
  - name: playwright
    command: playwright
    description: Rendered UI automation helper.
    when: Use when a UI task needs rendered page inspection.
    examples:
      - playwright test
```

Pass `--local` to scope a tool to the current project — it is written to `squire.yaml` instead and only surfaces in guides generated under that directory:

```bash
squire cli add game-krunker-harness --local --description "macOS dev harness." --when "Drive the running harness."
squire cli remove game-krunker-harness --local
```

`squire cli list` shows both scopes, prefixed with `[global]` or `[local]`. Local tools override component- and global-provided tools with the same name.

## Development

```bash
go test ./...
go fmt ./...
go build ./...
```
