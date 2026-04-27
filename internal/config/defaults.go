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
    - pg_hba.conf
    - postgresql.conf
    - "*.psql"
    - file: docker-compose.yml
      contains: postgres
    - file: docker-compose.yaml
      contains: postgres
    - file: compose.yml
      contains: postgres
    - file: compose.yaml
      contains: postgres
    - file: package.json
      regex: '"(pg|postgres|postgresql|node-postgres)"\s*:'
guidance:
  technologies:
    - "Persistence uses PostgreSQL."
  environment:
    - "DATABASE_URL or PG* env vars configure PostgreSQL connections."
  verification:
    - "Database changes need migration/reset verification when scripts exist."
`,
	},
	{
		Name: "mysql.yaml",
		Body: `version: 1
id: mysql
description: MySQL/MariaDB guidance.
detectors:
  any:
    - my.cnf
    - mysql.cnf
    - mariadb.cnf
    - file: docker-compose.yml
      regex: 'image:\s*["'']?(mysql|mariadb|percona)'
    - file: docker-compose.yaml
      regex: 'image:\s*["'']?(mysql|mariadb|percona)'
    - file: compose.yml
      regex: 'image:\s*["'']?(mysql|mariadb|percona)'
    - file: compose.yaml
      regex: 'image:\s*["'']?(mysql|mariadb|percona)'
    - file: package.json
      regex: '"(mysql|mysql2|mariadb)"\s*:'
guidance:
  technologies:
    - "Persistence uses MySQL."
  environment:
    - "MYSQL_* env vars configure MySQL connections."
  verification:
    - "Database changes need migration/reset verification when scripts exist."
`,
	},
	{
		Name: "mongodb.yaml",
		Body: `version: 1
id: mongodb
description: MongoDB guidance.
detectors:
  any:
    - mongod.conf
    - mongodb.conf
    - file: docker-compose.yml
      regex: 'image:\s*["'']?mongo'
    - file: docker-compose.yaml
      regex: 'image:\s*["'']?mongo'
    - file: compose.yml
      regex: 'image:\s*["'']?mongo'
    - file: compose.yaml
      regex: 'image:\s*["'']?mongo'
    - file: package.json
      regex: '"(mongodb|mongoose)"\s*:'
    - file: go.mod
      regex: 'go\.mongodb\.org/mongo-driver'
guidance:
  technologies:
    - "Document store uses MongoDB."
  environment:
    - "MONGO_URL or MONGODB_URI configures MongoDB connections."
`,
	},
	{
		Name: "redis.yaml",
		Body: `version: 1
id: redis
description: Redis guidance.
detectors:
  any:
    - redis.conf
    - file: docker-compose.yml
      regex: 'image:\s*["'']?redis'
    - file: docker-compose.yaml
      regex: 'image:\s*["'']?redis'
    - file: compose.yml
      regex: 'image:\s*["'']?redis'
    - file: compose.yaml
      regex: 'image:\s*["'']?redis'
    - file: package.json
      regex: '"(redis|ioredis)"\s*:'
    - file: go.mod
      regex: 'github\.com/(go-redis|redis)/'
guidance:
  technologies:
    - "Cache/queue uses Redis."
  environment:
    - "REDIS_URL configures Redis connections."
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
