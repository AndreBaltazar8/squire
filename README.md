# Squire

Squire normalizes the project instruction files used by coding agents. It gives a project consistent markerless `AGENTS.md` and `CLAUDE.md` files, keeps them analyzable by stable headings, and stores reusable agent/tool preferences under `~/.config/squire`.

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
go run ./cmd/squire detect

# Generate AGENTS.md and CLAUDE.md in the current project
go run ./cmd/squire generate all

# Select reusable guidance for a new/empty project
go run ./cmd/squire generate all -i --project-name myapp

# Generate with explicit components
go run ./cmd/squire generate all --component svelte --component go-api --project-name myapp

# Preview output without writing
go run ./cmd/squire generate all --stdout

# Download public component definitions
go run ./cmd/squire download AndreBaltazar8/squire-components
go run ./cmd/squire download AndreBaltazar8/squire-components#svelte

# Browse public component providers
go run ./cmd/squire browse
go run ./cmd/squire browse -i
go run ./cmd/squire browse AndreBaltazar8/squire-components#svelte

# Manage installed components
go run ./cmd/squire component list
go run ./cmd/squire component remove svelte

# Check section coverage in existing files
go run ./cmd/squire analyze

# Manage CLI tools that generated guides expose to agents
go run ./cmd/squire cli list
go run ./cmd/squire cli add playwright --description "Rendered UI helper." --when "Use for rendered UI verification." --example "playwright test"
go run ./cmd/squire cli remove playwright
```

By default, Squire auto-detects components from current project files. `--component <id>` selects reusable guidance for that generation, and `generate -i` opens a searchable selector. Squire does not write project-local metadata; rerun with explicit components when a choice cannot be detected from files.

## Markerless Guides

Generated `AGENTS.md` and `CLAUDE.md` files are plain Markdown with stable section headings.

`squire analyze` uses headings to detect missing required sections. Regeneration also uses headings to preserve the `Squire Custom Notes` section and to decide whether an existing markerless guide has Squire's expected shape.

Generated `AGENTS.md` includes a `Squire Custom Notes` section. Add project-specific notes there; Squire preserves that section across regeneration.

## Configuration

Squire creates its global config file on first use:

- `~/.config/squire/config.yaml`

Optional components live in `~/.config/squire/components/*.yaml`. Components can detect files or be selected during generation, then add guidance, commands, and CLI tools to generated agent files.

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

## Development

```bash
go test ./...
go fmt ./...
go build ./...
```
