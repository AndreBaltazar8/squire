package project

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/AndreBaltazar8/squire/internal/config"
)

func TestSelectedComponentsWorkForEmptyProject(t *testing.T) {
	dir := t.TempDir()

	components := []config.Component{
		{
			ID: "svelte",
			Guidance: config.ComponentGuidance{
				Technologies: []string{"Frontend uses SvelteKit."},
				Structure:    []string{"`src/site-*` holds SvelteKit apps."},
				Commands:     []config.ComponentCommand{{Command: "bun run dev", Description: "local dev"}},
			},
		},
		{
			ID: "deploy",
			Guidance: config.ComponentGuidance{
				Commands: []config.ComponentCommand{{Command: "deployctl dev", Description: "local dev"}},
			},
		},
	}

	info, err := Detect(dir, "demo", []string{"svelte", "deploy"}, components, config.ProjectConfig{})
	if err != nil {
		t.Fatal(err)
	}

	if !contains(info.Components, "svelte") || !contains(info.Components, "deploy") {
		t.Fatalf("components = %#v", info.Components)
	}
	if !contains(info.Structure, "`src/site-*` holds SvelteKit apps.") {
		t.Fatalf("structure = %#v", info.Structure)
	}
	if !containsCommand(info.Commands, "bun run dev") {
		t.Fatalf("commands = %#v", info.Commands)
	}
	if !contains(info.Technologies, "Frontend uses SvelteKit.") {
		t.Fatalf("technologies = %#v", info.Technologies)
	}
}

func TestDetectSrcSubprojects(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "src", "svc-api", "go.mod"), "module example.com/api\n\ngo 1.25\n")
	mustWrite(t, filepath.Join(dir, "src", "site-web", "package.json"), `{
  "name": "web",
  "scripts": {
    "dev": "vite dev",
    "build": "vite build",
    "lint": "eslint ."
  },
  "dependencies": {
    "@sveltejs/kit": "latest",
    "vite": "latest"
  },
  "devDependencies": {
    "typescript": "latest"
  }
}`)

	info, err := Detect(dir, "demo", nil, nil, config.ProjectConfig{})
	if err != nil {
		t.Fatal(err)
	}

	if !contains(info.Structure, "`src/svc-api` is Go service `example.com/api`.") {
		t.Fatalf("structure = %#v", info.Structure)
	}
	if !contains(info.Structure, "`src/site-web` is SvelteKit app `web` with scripts `dev`, `build`, `lint`.") {
		t.Fatalf("structure = %#v", info.Structure)
	}
}

func TestDetectedComponentAddsGuidanceAndTools(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "deploy.yaml"), "{}")

	components := []config.Component{
		{
			ID: "deploy",
			Detectors: config.ComponentDetectors{
				Any: []config.DetectorRule{{Glob: "deploy.yaml"}},
			},
			Guidance: config.ComponentGuidance{
				Commands: []config.ComponentCommand{{Command: "deployctl dev", Description: "local dev"}},
				Workflow: []string{"Use `deployctl dev` for local project environment."},
			},
			CLITools: []config.ToolConfig{{Name: "deployctl", Command: "deployctl"}},
		},
	}

	info, err := Detect(dir, "demo", nil, components, config.ProjectConfig{})
	if err != nil {
		t.Fatal(err)
	}

	if !containsCommand(info.Commands, "deployctl dev") {
		t.Fatalf("commands = %#v", info.Commands)
	}
	if len(info.Tools) != 1 || info.Tools[0].Name != "deployctl" {
		t.Fatalf("tools = %#v", info.Tools)
	}
	if !contains(info.Workflow, "Use `deployctl dev` for local project environment.") {
		t.Fatalf("workflow = %#v", info.Workflow)
	}
}

func TestProjectConfigDescriptionOverridesReadme(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "README.md"), "# Setup\n\nRun `make build` first.\n")

	override := "Demo is a tiny in-memory key/value store used in tests."
	info, err := Detect(dir, "demo", nil, nil, config.ProjectConfig{Description: override})
	if err != nil {
		t.Fatal(err)
	}

	if !contains(info.Overview, override) {
		t.Fatalf("overview = %#v (expected to contain %q)", info.Overview, override)
	}
	if contains(info.Overview, "Run `make build` first.") {
		t.Fatalf("overview unexpectedly fell back to README paragraph: %#v", info.Overview)
	}
}

func TestDetectDesignFileAddsGuidance(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "DESIGN.md"), "# Design System\n")

	info, err := Detect(dir, "demo", nil, nil, config.ProjectConfig{})
	if err != nil {
		t.Fatal(err)
	}

	if !contains(info.DesignFiles, "DESIGN.md") {
		t.Fatalf("design files = %#v", info.DesignFiles)
	}
	if !contains(info.Structure, "`DESIGN.md` contains visual design system.") {
		t.Fatalf("structure = %#v", info.Structure)
	}
	if !contains(info.Design, "`DESIGN.md` is the visual design system source of truth. Read before frontend/UI changes.") {
		t.Fatalf("design = %#v", info.Design)
	}
	if !contains(info.Design, "When changing root `DESIGN.md`, run `npx @google/design.md lint DESIGN.md` when available.") {
		t.Fatalf("design = %#v", info.Design)
	}
}

func TestStructureDescribesMonorepoTopLevelDirs(t *testing.T) {
	dir := t.TempDir()
	for _, sub := range []string{"pkg-go", "infra", "k8s", "helm", "docker", "specs", "db", "migrations", "terraform"} {
		mustWrite(t, filepath.Join(dir, sub, ".keep"), "")
	}
	mustWrite(t, filepath.Join(dir, "Makefile"), "build:\n")
	mustWrite(t, filepath.Join(dir, "Cargo.toml"), "[package]\nname = \"x\"\n")

	info, err := Detect(dir, "demo", nil, nil, config.ProjectConfig{})
	if err != nil {
		t.Fatal(err)
	}

	wants := []string{
		"`pkg-go/` contains shared Go packages.",
		"`infra/` contains infrastructure-as-code.",
		"`k8s/` contains Kubernetes manifests.",
		"`helm/` contains Helm chart values.",
		"`docker/` contains Dockerfiles.",
		"`specs/` contains feature specs and RFCs.",
		"`db/` contains database schema and tooling.",
		"`migrations/` contains database migrations.",
		"`terraform/` contains Terraform IaC.",
		"`Makefile` contains Make targets.",
		"`Cargo.toml` contains Rust manifest.",
	}
	for _, want := range wants {
		if !contains(info.Structure, want) {
			t.Fatalf("missing %q in structure:\n%v", want, info.Structure)
		}
	}
}

func TestSelectedComponentWorksWithoutDetectorMatch(t *testing.T) {
	dir := t.TempDir()

	components := []config.Component{
		{
			ID: "deploy",
			Guidance: config.ComponentGuidance{
				Commands: []config.ComponentCommand{{Command: "deployctl ship", Description: "deploy"}},
			},
		},
	}

	info, err := Detect(dir, "demo", []string{"deploy"}, components, config.ProjectConfig{})
	if err != nil {
		t.Fatal(err)
	}

	if !containsCommand(info.Commands, "deployctl ship") {
		t.Fatalf("commands = %#v", info.Commands)
	}
}

func TestGamePlatformMonorepoDetection(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "README.md"), "# Game Platform\n\nGame Platform is a first vertical slice for a web multiplayer game platform:\n")
	mustWrite(t, filepath.Join(dir, "Makefile"), "build:\n\ntest:\n")
	mustWrite(t, filepath.Join(dir, "package.json"), `{
  "name": "game-platform",
  "packageManager": "bun@1.3.0",
  "scripts": {
    "build": "make build",
    "dev": "bash scripts/dev.sh",
    "e2e": "bun run scripts/e2e.mjs",
    "test": "make test"
  }
}`)
	mustWrite(t, filepath.Join(dir, "src", "app-client", "package.json"), `{
  "name": "@game/app-client",
  "scripts": {"dev": "vite", "build": "vite build"},
  "dependencies": {"three": "^0.182.0"},
  "devDependencies": {"vite": "^7.2.7", "typescript": "^5.9.3"}
}`)
	mustWrite(t, filepath.Join(dir, "src", "app-client", "cpp", "client_bridge.cpp"), "int main() { return 0; }\n")
	mustWrite(t, filepath.Join(dir, "src", "lib-engine", "src", "engine.cpp"), "int engine() { return 0; }\n")
	mustWrite(t, filepath.Join(dir, "src", "lib-engine", "tests", "engine_tests.cpp"), "int main() { return 0; }\n")
	mustWrite(t, filepath.Join(dir, "src", "svc-game-server", "src", "main.cpp"), "int main() { return 0; }\n")
	mustWrite(t, filepath.Join(dir, "src", "svc-api", "go.mod"), "module game/svc-api\n\ngo 1.25\n\nrequire github.com/gofiber/fiber/v3 v3.1.0\n")

	info, err := Detect(dir, "game-platform", nil, nil, config.ProjectConfig{})
	if err != nil {
		t.Fatal(err)
	}

	if containsPrefix(info.Technologies, "Frontend: SvelteKit") {
		t.Fatalf("unexpected SvelteKit component guidance = %#v", info.Technologies)
	}
	if !contains(info.Structure, "`src/app-client` is Vite + Three.js app `@game/app-client`, scripts `dev`, `build`, with C++/WASM bridge.") {
		t.Fatalf("structure = %#v", info.Structure)
	}
	if !contains(info.Structure, "`src/lib-engine` is C++ library with tests.") {
		t.Fatalf("structure = %#v", info.Structure)
	}
	if !contains(info.Structure, "`src/svc-game-server` is C++ service.") {
		t.Fatalf("structure = %#v", info.Structure)
	}
	if !contains(info.Structure, "`src/svc-api` is Go/Fiber API `game/svc-api`.") {
		t.Fatalf("structure = %#v", info.Structure)
	}
	if !containsCommand(info.Commands, "bun run e2e") || !containsCommand(info.Commands, "make build") {
		t.Fatalf("commands = %#v", info.Commands)
	}
}

func TestMixedSvelteGoWorkspaceDetection(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "package.json"), `{
  "name": "mixed-workspace",
  "packageManager": "bun@1.3.4",
  "workspaces": ["src/*"],
  "scripts": {"build": "turbo run build", "dev": "turbo run dev", "db:reset": "bash scripts/reset-local-postgres.sh"},
  "devDependencies": {"turbo": "^1.10.14"}
}`)
	mustWrite(t, filepath.Join(dir, "src", "pkg-ts-game", "package.json"), `{
  "name": "@example/pkg-ts-game",
  "scripts": {"build": "tsc -p tsconfig.json --noEmit", "check": "tsc -p tsconfig.json --noEmit"},
  "devDependencies": {"typescript": "^5.9.3"}
}`)
	mustWrite(t, filepath.Join(dir, "src", "site-web", "package.json"), `{
  "name": "@example/site-web",
  "scripts": {"dev": "vite dev", "build": "vite build", "check": "svelte-kit sync && svelte-check"},
  "dependencies": {"three": "^0.172.0"},
  "devDependencies": {"@sveltejs/kit": "^2.50.0", "vite": "^7.3.1", "typescript": "^5.9.3"}
}`)
	mustWrite(t, filepath.Join(dir, "src", "svc-gameserver", "package.json"), `{
  "name": "@example/svc-gameserver",
  "scripts": {"dev": "bun run --watch src/index.ts", "build": "bun build src/index.ts --target bun", "test": "bun test"},
  "dependencies": {"jose": "^6.2.2"}
}`)
	mustWrite(t, filepath.Join(dir, "src", "svc-api", "go.mod"), "module example/svc-api\n\ngo 1.25\n\nrequire github.com/gofiber/fiber/v3 v3.1.0\n")

	info, err := Detect(dir, "mixed-workspace", nil, nil, config.ProjectConfig{})
	if err != nil {
		t.Fatal(err)
	}

	if !contains(info.Structure, "`src/pkg-ts-game` is TypeScript package `@example/pkg-ts-game` with scripts `build`, `check`.") {
		t.Fatalf("structure = %#v", info.Structure)
	}
	if !contains(info.Structure, "`src/site-web` is SvelteKit app `@example/site-web` with scripts `dev`, `build`, `check`.") {
		t.Fatalf("structure = %#v", info.Structure)
	}
	if !contains(info.Structure, "`src/svc-gameserver` is service `@example/svc-gameserver` with scripts `dev`, `build`, `test`.") {
		t.Fatalf("structure = %#v", info.Structure)
	}
	if !containsCommand(info.Commands, "bun run db:reset") {
		t.Fatalf("commands = %#v", info.Commands)
	}
}

func mustWrite(t *testing.T, path string, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func contains(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}

func containsPrefix(items []string, prefix string) bool {
	for _, item := range items {
		if strings.HasPrefix(item, prefix) {
			return true
		}
	}
	return false
}

func containsCommand(commands []Command, want string) bool {
	for _, command := range commands {
		if command.Command == want {
			return true
		}
	}
	return false
}
