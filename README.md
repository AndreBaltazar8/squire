# Squire

Squire normalizes the project instruction files used by coding agents. It gives a project a consistent `AGENTS.md` and `CLAUDE.md`, keeps those files analyzable with Squire section markers, and stores reusable agent/tool preferences under `~/.config/squire`.

## Why

Different agents read different files:

- Codex and other tools commonly use `AGENTS.md`.
- Claude Code reads `CLAUDE.md`.
- Cursor Agent is configured by project rules, but can still benefit from a shared project guide.

Squire treats `AGENTS.md` as the shared source of truth and generates `CLAUDE.md` as a thin Claude-specific wrapper that imports it.

It detects root project files and immediate `src/*` subprojects, including Go services and SvelteKit/Vite apps, and writes concise project-structure phrases.

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
go run ./cmd/squire browse AndreBaltazar8/squire-components#svelte

# Check section coverage in existing files
go run ./cmd/squire analyze

# Manage CLI tools that generated guides expose to agents
go run ./cmd/squire cli list
go run ./cmd/squire cli add playwright --description "Rendered UI helper." --when "Use for rendered UI verification." --example "playwright test"
go run ./cmd/squire cli remove playwright
```

By default, Squire auto-detects components from project files. `--component <id>` selects reusable guidance manually, and `generate -i` opens a searchable selector. Applied component IDs are saved in the managed comment so later generations use the same set.

## Squire Markers

Generated sections are wrapped in HTML comments:

```markdown
<!-- squire:start id=technology-stack required=true -->
## Technology Stack

- Go
<!-- squire:end id=technology-stack -->
```

`squire analyze` uses those markers to detect missing required sections. For Claude Code, HTML comments are stripped before context injection, so the markers are maintenance metadata rather than model-facing guidance.

Generated `AGENTS.md` includes a `Squire Custom Notes` section. Add project-specific notes there; Squire preserves that section across regeneration.

## Configuration

Squire creates its global config file on first use:

- `~/.config/squire/config.yaml`

Optional components live in `~/.config/squire/components/*.yaml`. Components can detect files or be selected during generation, then add guidance, commands, and CLI tools to generated agent files.

Use `squire browse` to list public providers from `AndreBaltazar8/squire-components`. Use `squire download <owner>/<repo>` to install all components from a GitHub provider, or append `#component-id` to install one. A provider can define `provider.yaml` with `components: <dir>`; otherwise Squire looks for `components/`.

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
