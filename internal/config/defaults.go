package config

const DefaultConfigYAML = `version: 1
agents:
  - name: agent
    command: agent
    description: Cursor Agent CLI
    files:
      - AGENTS.md
      - .cursor/rules/*.mdc
  - name: codex
    command: codex
    description: OpenAI Codex CLI
    files:
      - AGENTS.md
  - name: claude
    command: claude
    description: Claude Code CLI
    files:
      - CLAUDE.md
      - AGENTS.md
`

type DefaultComponent struct {
	Name string
	Body string
}

var DefaultComponents = []DefaultComponent{
	{
		Name: "bun-workspace.yaml",
		Body: `version: 1
id: bun-workspace
description: Bun workspace and Turbo guidance.
detectors:
  any:
    - bun.lock
    - bun.lockb
guidance:
  technologies:
    - "Bun manages workspace scripts and dependencies."
  commands:
    - command: bun install
      description: install dependencies
    - command: bun run dev
      description: local development
    - command: bun run build
      description: production build
    - command: bun run test
      description: tests
  standards:
    - "Use existing package scripts before adding new commands."
`,
	},
	{
		Name: "go.yaml",
		Body: `version: 1
id: go
description: Go project guidance.
detectors:
  any:
    - go.mod
guidance:
  technologies:
    - "Go project."
  commands:
    - command: go test ./...
      description: Go tests
    - command: go fmt ./...
      description: Go format
    - command: go build ./...
      description: Go build
  standards:
    - "Go code follows existing package boundaries."
  verification:
    - "Run focused Go tests for touched packages."
`,
	},
	{
		Name: "go-api.yaml",
		Body: `version: 1
id: go-api
description: Go service guidance.
detectors:
  any:
    - src/svc-*/go.mod
guidance:
  technologies:
    - "Backend uses Go services in src/svc-*."
  structure:
    - "src/svc-* holds Go services."
  standards:
    - "Go code follows existing package boundaries."
  verification:
    - "Run Go checks from touched service directory."
`,
	},
	{
		Name: "postgres.yaml",
		Body: `version: 1
id: postgres
description: PostgreSQL guidance.
detectors:
  any:
    - docker-compose.yml
    - compose.yaml
    - migrations
    - db
guidance:
  technologies:
    - "Persistence uses PostgreSQL when configured."
  environment:
    - "DATABASE_URL configures PostgreSQL connections."
  verification:
    - "Database changes need migration/reset verification when scripts exist."
`,
	},
	{
		Name: "svelte.yaml",
		Body: `version: 1
id: svelte
description: SvelteKit project guidance.
detectors:
  any:
    - svelte.config.js
    - svelte.config.ts
    - src/site-*/svelte.config.js
    - src/site-*/svelte.config.ts
guidance:
  technologies:
    - "Frontend uses SvelteKit."
  structure:
    - "src/site-* holds SvelteKit apps."
  standards:
    - "Svelte code follows existing route/component layout."
  verification:
    - "UI changes need rendered check when layout matters."
`,
	},
}
