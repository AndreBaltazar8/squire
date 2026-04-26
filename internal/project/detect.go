package project

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"squire/internal/config"
)

type Info struct {
	Name         string              `json:"name"`
	Root         string              `json:"root"`
	Components   []string            `json:"components"`
	Overview     []string            `json:"overview"`
	Technologies []string            `json:"technologies"`
	Structure    []string            `json:"structure"`
	Commands     []Command           `json:"commands"`
	Environment  []string            `json:"environment"`
	Workflow     []string            `json:"workflow"`
	Standards    []string            `json:"standards"`
	Verification []string            `json:"verification"`
	Tools        []config.ToolConfig `json:"tools"`
}

type Command struct {
	Command     string `json:"command"`
	Description string `json:"description"`
}

type packageJSON struct {
	Name            string            `json:"name"`
	PackageManager  string            `json:"packageManager"`
	Scripts         map[string]string `json:"scripts"`
	Dependencies    map[string]string `json:"dependencies"`
	DevDependencies map[string]string `json:"devDependencies"`
}

func Detect(root, nameOverride string, selectedComponents []string, components []config.Component) (Info, error) {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return Info{}, err
	}

	name := nameOverride
	if name == "" {
		name = filepath.Base(absRoot)
	}

	info := Info{Name: name, Root: absRoot}
	info.Overview = detectOverview(absRoot, name)
	info.Technologies = detectTechnologies(absRoot)
	info.Structure = detectStructure(absRoot)
	info.Commands = detectCommands(absRoot)
	info.Environment = defaultEnvironment()
	info.Workflow = defaultWorkflow()
	info.Standards = defaultStandards()
	info.Verification = defaultVerification()

	applyComponents(&info, absRoot, selectedComponents, components)

	if len(info.Technologies) == 0 {
		info.Technologies = []string{"No technology stack detected yet. Regenerate with `--component <id>` or edit this guide with project-specific stack details."}
	}
	if len(info.Structure) == 0 {
		info.Structure = []string{"No project files detected yet."}
	}
	if len(info.Commands) == 0 {
		info.Commands = []Command{{Command: "Add project commands", Description: "Document setup, development, test, lint, and build commands for this project."}}
	}

	return info, nil
}

func applyComponents(info *Info, root string, selectedComponents []string, components []config.Component) {
	selected := selectedComponentSet(selectedComponents)
	for _, component := range components {
		if component.ID == "" {
			continue
		}
		if selectedComponents == nil {
			if !componentMatches(root, component) {
				continue
			}
		} else if !selected[component.ID] {
			continue
		}
		applyComponent(info, component)
		info.Components = append(info.Components, component.ID)
	}

	info.Components = dedupe(info.Components)
	info.Overview = dedupe(info.Overview)
	info.Technologies = dedupe(info.Technologies)
	info.Structure = dedupe(info.Structure)
	info.Commands = dedupeCommands(info.Commands)
	info.Environment = dedupe(info.Environment)
	info.Workflow = dedupe(info.Workflow)
	info.Standards = dedupe(info.Standards)
	info.Verification = dedupe(info.Verification)
	info.Tools = dedupeTools(info.Tools)
}

func selectedComponentSet(ids []string) map[string]bool {
	out := map[string]bool{}
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id != "" {
			out[id] = true
		}
	}
	return out
}

func DetectedComponentIDs(root string, components []config.Component) []string {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil
	}
	var ids []string
	for _, component := range components {
		if component.ID != "" && componentMatches(absRoot, component) {
			ids = append(ids, component.ID)
		}
	}
	return dedupe(ids)
}

func applyComponent(info *Info, component config.Component) {
	info.Overview = append(info.Overview, component.Guidance.Overview...)
	info.Technologies = append(info.Technologies, component.Guidance.Technologies...)
	info.Structure = append(info.Structure, component.Guidance.Structure...)
	for _, command := range component.Guidance.Commands {
		info.Commands = append(info.Commands, Command{
			Command:     command.Command,
			Description: command.Description,
		})
	}
	info.Environment = append(info.Environment, component.Guidance.Environment...)
	info.Workflow = append(info.Workflow, component.Guidance.Workflow...)
	info.Standards = append(info.Standards, component.Guidance.Standards...)
	info.Verification = append(info.Verification, component.Guidance.Verification...)
	info.Tools = append(info.Tools, component.CLITools...)
}

func componentMatches(root string, component config.Component) bool {
	detectors := component.Detectors
	if len(detectors.Any) == 0 && len(detectors.All) == 0 {
		return false
	}
	if len(detectors.All) > 0 {
		for _, pattern := range detectors.All {
			if !hasGlob(root, pattern) {
				return false
			}
		}
		return true
	}
	for _, pattern := range detectors.Any {
		if hasGlob(root, pattern) {
			return true
		}
	}
	return false
}

func detectOverview(root, name string) []string {
	readme := readFirstExisting(root, "README.md", "readme.md", "README")
	if readme != "" {
		if summary := firstReadableParagraph(readme); summary != "" {
			summary = strings.TrimSuffix(summary, ":")
			return []string{summary}
		}
	}
	return []string{"`" + name + "` needs purpose and key workflows documented here."}
}

func detectTechnologies(root string) []string {
	var tech []string

	if body := readFile(root, "go.mod"); body != "" {
		module := parseGoModule(body)
		version := parseGoVersion(body)
		item := "Go"
		if version != "" {
			item += " " + version
		}
		if module != "" {
			item += " module `" + module + "`"
		}
		tech = append(tech, item)
	}

	for _, path := range glob(root, "src/svc-*/go.mod") {
		body := readFile(root, path)
		module := parseGoModule(body)
		item := "Go service"
		if module != "" {
			item += " module `" + module + "`"
		}
		tech = append(tech, item)
	}

	if pkg := readPackageJSON(root); pkg != nil {
		tech = append(tech, packageTechnologies(root, pkg)...)
	}

	if exists(root, "Makefile") {
		tech = append(tech, "Make build/test orchestration")
	}
	if hasGlob(root, "src/*/**/*.cpp") || hasGlob(root, "src/*/**/*.hpp") || hasGlob(root, "src/*/**/*.h") {
		tech = append(tech, "C++")
	}
	if hasGlob(root, "src/app-*/cpp/*.cpp") {
		tech = append(tech, "Emscripten/WASM bridge")
	}

	for _, path := range glob(root, "src/site-*/package.json") {
		dir := filepath.Dir(path)
		pkg := readPackageJSONAt(root, path)
		if pkg == nil {
			continue
		}
		for _, item := range packageTechnologies(filepath.Join(root, dir), pkg) {
			tech = append(tech, item)
		}
	}

	if exists(root, "pyproject.toml") {
		tech = append(tech, "Python project (`pyproject.toml`)")
	}
	if exists(root, "Cargo.toml") {
		tech = append(tech, "Rust project (`Cargo.toml`)")
	}
	if exists(root, "Dockerfile") || exists(root, "docker-compose.yml") || exists(root, "compose.yaml") {
		tech = append(tech, "Docker")
	}
	if exists(root, ".cursor/rules") {
		tech = append(tech, "Cursor project rules")
	}

	return dedupe(tech)
}

func packageTechnologies(root string, pkg *packageJSON) []string {
	var tech []string
	pm := detectPackageManager(root, pkg.PackageManager)
	if pm != "" {
		tech = append(tech, pm+" package management")
	} else {
		tech = append(tech, "JavaScript/TypeScript package workspace")
	}

	deps := mergeStringMaps(pkg.Dependencies, pkg.DevDependencies)
	if _, ok := deps["@sveltejs/kit"]; ok {
		tech = append(tech, "SvelteKit")
	}
	if _, ok := deps["vite"]; ok {
		tech = append(tech, "Vite")
	}
	if _, ok := deps["three"]; ok {
		tech = append(tech, "Three.js")
	}
	if _, ok := deps["typescript"]; ok {
		tech = append(tech, "TypeScript")
	}
	if _, ok := deps["turbo"]; ok {
		tech = append(tech, "Turbo")
	}
	if _, ok := deps["react"]; ok {
		tech = append(tech, "React")
	}
	if _, ok := deps["vue"]; ok {
		tech = append(tech, "Vue")
	}
	return dedupe(tech)
}

func detectStructure(root string) []string {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil
	}

	var lines []string
	srcProjectCount := 0
	for _, entry := range entries {
		name := entry.Name()
		if strings.HasPrefix(name, ".git") || name == "node_modules" {
			continue
		}
		if name == "src" {
			srcProjectCount = len(detectSrcSubprojects(root))
		}
		if name == "src" && srcProjectCount > 0 {
			continue
		}
		if entry.IsDir() {
			if purpose := describePath(name, true); purpose != "" {
				lines = append(lines, sentenceForPath(name+"/", purpose))
			}
			continue
		}
		if isImportantRootFile(name) {
			lines = append(lines, sentenceForPath(name, describePath(name, false)))
		}
	}
	sort.Strings(lines)
	if srcProjectCount > 0 {
		lines = append(lines, detectSrcSubprojects(root)...)
	}
	if len(lines) > 20 {
		lines = lines[:20]
	}
	return lines
}

func detectSrcSubprojects(root string) []string {
	src := filepath.Join(root, "src")
	entries, err := os.ReadDir(src)
	if err != nil {
		return nil
	}

	var lines []string
	for _, entry := range entries {
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") || entry.Name() == "node_modules" {
			continue
		}
		rel := filepath.Join("src", entry.Name())
		if line := describeSubproject(root, rel); line != "" {
			lines = append(lines, line)
		}
	}
	sort.Strings(lines)
	if len(lines) > 12 {
		remaining := len(lines) - 12
		lines = append(lines[:12], itoa(remaining)+" more `src/*` projects exist.")
	}
	return lines
}

func describeSubproject(root, rel string) string {
	path := filepath.ToSlash(rel)
	name := filepath.Base(rel)
	var parts []string

	if body := readFile(root, filepath.Join(rel, "go.mod")); body != "" {
		kind := "Go module"
		if strings.HasPrefix(name, "svc-") {
			kind = "Go service"
		}
		if strings.HasPrefix(name, "svc-") && hasGoDependency(body, "github.com/gofiber/fiber") {
			kind = "Go/Fiber API"
		}
		if module := parseGoModule(body); module != "" {
			parts = append(parts, kind+" `"+module+"`")
		} else {
			parts = append(parts, kind)
		}
	}

	if pkg := readPackageJSONAt(root, filepath.Join(rel, "package.json")); pkg != nil {
		parts = append(parts, describePackageSubproject(name, pkg))
		if scripts := importantScripts(pkg.Scripts); len(scripts) > 0 {
			parts = append(parts, "scripts `"+strings.Join(scripts, "`, `")+"`")
		}
	}

	if hasGlob(root, filepath.Join(rel, "**/*.cpp")) || hasGlob(root, filepath.Join(rel, "**/*.hpp")) || hasGlob(root, filepath.Join(rel, "**/*.h")) {
		parts = append(parts, describeCXXSubproject(name, root, rel))
	}

	if len(parts) == 0 {
		switch {
		case strings.HasPrefix(name, "svc-"):
			parts = append(parts, "service")
		case strings.HasPrefix(name, "site-"):
			parts = append(parts, "site")
		default:
			parts = append(parts, "source project")
		}
	}

	return "`" + path + "` is " + joinPhrase(parts) + "."
}

func describePackageSubproject(name string, pkg *packageJSON) string {
	deps := mergeStringMaps(pkg.Dependencies, pkg.DevDependencies)
	kind := "JS package"
	switch {
	case hasDependency(deps, "@sveltejs/kit"):
		kind = "SvelteKit app"
	case hasDependency(deps, "three"):
		kind = "Vite + Three.js app"
	case hasDependency(deps, "react"):
		kind = "React app"
	case hasDependency(deps, "vue"):
		kind = "Vue app"
	case hasDependency(deps, "vite"):
		kind = "Vite app"
	case strings.HasPrefix(name, "site-"):
		kind = "site"
	case strings.HasPrefix(name, "svc-"):
		kind = "service"
	case strings.HasPrefix(name, "pkg-"):
		kind = "TypeScript package"
	}
	if pkg.Name != "" {
		return kind + " `" + pkg.Name + "`"
	}
	return kind
}

func describeCXXSubproject(name, root, rel string) string {
	switch {
	case strings.HasPrefix(name, "svc-"):
		if hasGlob(root, filepath.Join(rel, "src/main.cpp")) {
			return "C++ service"
		}
		return "C++ service code"
	case strings.HasPrefix(name, "lib-"):
		if hasGlob(root, filepath.Join(rel, "tests/*.cpp")) {
			return "C++ library with tests"
		}
		return "C++ library"
	case hasGlob(root, filepath.Join(rel, "cpp/*.cpp")):
		return "C++/WASM bridge"
	default:
		return "C++ code"
	}
}

func hasGoDependency(body, dependencyPrefix string) bool {
	for _, line := range strings.Split(body, "\n") {
		if strings.Contains(line, dependencyPrefix) {
			return true
		}
	}
	return false
}

func importantScripts(scripts map[string]string) []string {
	order := []string{"dev", "build", "test", "lint", "check", "e2e", "preview", "start", "db:reset"}
	var out []string
	for _, name := range order {
		if _, ok := scripts[name]; ok {
			out = append(out, name)
		}
	}
	return out
}

func sentenceForPath(path, purpose string) string {
	switch purpose {
	case "overview", "docs", "secrets":
		return "`" + path + "` contains " + purpose + "."
	case "directory", "file":
		return "`" + path + "` is project " + purpose + "."
	default:
		return "`" + path + "` contains " + purpose + "."
	}
}

func joinPhrase(parts []string) string {
	switch len(parts) {
	case 0:
		return ""
	case 1:
		return parts[0]
	case 2:
		return parts[0] + " with " + parts[1]
	default:
		return strings.Join(parts[:len(parts)-1], ", ") + ", with " + parts[len(parts)-1]
	}
}

func hasDependency(deps map[string]string, name string) bool {
	_, ok := deps[name]
	return ok
}

func detectCommands(root string) []Command {
	var commands []Command
	if exists(root, "go.mod") {
		commands = append(commands,
			Command{Command: "go test ./...", Description: "Go tests"},
			Command{Command: "go fmt ./...", Description: "Go format"},
			Command{Command: "go build ./...", Description: "Go build"},
		)
	}

	if pkg := readPackageJSON(root); pkg != nil {
		pm := commandPackageManager(root, pkg.PackageManager)
		for _, name := range []string{"dev", "build", "test", "lint", "check", "format", "e2e", "preview", "start", "db:reset"} {
			if _, ok := pkg.Scripts[name]; ok {
				commands = append(commands, Command{
					Command:     pm + " run " + name,
					Description: "`" + name + "` package script",
				})
			}
		}
	}
	if exists(root, "Makefile") {
		commands = append(commands,
			Command{Command: "make build", Description: "Make build"},
			Command{Command: "make test", Description: "Make tests"},
		)
	}

	return dedupeCommands(commands)
}

func parseGoModule(body string) string {
	for _, line := range strings.Split(body, "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && fields[0] == "module" {
			return fields[1]
		}
	}
	return ""
}

func parseGoVersion(body string) string {
	for _, line := range strings.Split(body, "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && fields[0] == "go" {
			return fields[1]
		}
	}
	return ""
}

func readPackageJSON(root string) *packageJSON {
	return readPackageJSONAt(root, "package.json")
}

func readPackageJSONAt(root, path string) *packageJSON {
	body := readFile(root, path)
	if body == "" {
		return nil
	}
	var pkg packageJSON
	if err := json.Unmarshal([]byte(body), &pkg); err != nil {
		return nil
	}
	return &pkg
}

func detectPackageManager(root, declared string) string {
	if declared != "" {
		return displayPackageManager(strings.Split(declared, "@")[0])
	}
	if exists(root, "bun.lock") || exists(root, "bun.lockb") {
		return "Bun"
	}
	if exists(root, "pnpm-lock.yaml") {
		return "pnpm"
	}
	if exists(root, "yarn.lock") {
		return "Yarn"
	}
	if exists(root, "package-lock.json") {
		return "npm"
	}
	return ""
}

func displayPackageManager(value string) string {
	switch strings.ToLower(value) {
	case "bun":
		return "Bun"
	case "pnpm":
		return "pnpm"
	case "yarn":
		return "Yarn"
	case "npm":
		return "npm"
	default:
		return value
	}
}

func commandPackageManager(root, declared string) string {
	pm := strings.ToLower(detectPackageManager(root, declared))
	switch pm {
	case "bun":
		return "bun"
	case "pnpm":
		return "pnpm"
	case "yarn":
		return "yarn"
	default:
		return "npm"
	}
}

func mergeStringMaps(left, right map[string]string) map[string]string {
	out := map[string]string{}
	for key, val := range left {
		out[key] = val
	}
	for key, val := range right {
		out[key] = val
	}
	return out
}

func describePath(name string, dir bool) string {
	switch name {
	case "cmd":
		return "CLI entrypoints"
	case "internal":
		return "private Go packages"
	case "pkg":
		return "public packages"
	case "src":
		return "source projects"
	case "docs":
		return "docs"
	case "packages":
		return "workspace packages"
	case "secrets":
		return "secrets"
	case ".cursor":
		return "Cursor config"
	case ".claude":
		return "Claude config"
	case "scripts":
		return "scripts"
	case "go.mod":
		return "Go module"
	case "go.sum":
		return "Go lockfile"
	case "package.json":
		return "JS manifest"
	case "config.yaml":
		return "project config"
	case "README.md":
		return "overview"
	default:
		return ""
	}
}

func isImportantRootFile(name string) bool {
	switch name {
	case "go.mod", "go.sum", "package.json", "README.md", "readme.md", "config.yaml", "Dockerfile", "docker-compose.yml", "compose.yaml", "pyproject.toml", "Cargo.toml":
		return true
	default:
		return false
	}
}

func firstReadableParagraph(body string) string {
	lines := strings.Split(body, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "<!--") {
			continue
		}
		return line
	}
	return ""
}

func readFirstExisting(root string, names ...string) string {
	for _, name := range names {
		if body := readFile(root, name); body != "" {
			return body
		}
	}
	return ""
}

func readFile(root, name string) string {
	body, err := os.ReadFile(filepath.Join(root, name))
	if err != nil {
		return ""
	}
	return string(body)
}

func exists(root, name string) bool {
	_, err := os.Stat(filepath.Join(root, name))
	return err == nil
}

func hasGlob(root, pattern string) bool {
	return len(glob(root, pattern)) > 0
}

func dedupeTools(tools []config.ToolConfig) []config.ToolConfig {
	seen := map[string]bool{}
	var out []config.ToolConfig
	for _, tool := range tools {
		if tool.Name == "" || seen[tool.Name] {
			continue
		}
		seen[tool.Name] = true
		out = append(out, tool)
	}
	return out
}

func glob(root, pattern string) []string {
	matches, err := filepath.Glob(filepath.Join(root, pattern))
	if err != nil {
		return nil
	}
	out := make([]string, 0, len(matches))
	for _, match := range matches {
		rel, err := filepath.Rel(root, match)
		if err != nil {
			continue
		}
		out = append(out, rel)
	}
	sort.Strings(out)
	return out
}

func defaultEnvironment() []string {
	return []string{
		"Do not commit secrets unless encrypted workflow exists.",
		"Prefer documented project commands.",
		"Start smallest service environment needed.",
	}
}

func defaultWorkflow() []string {
	return []string{
		"Read relevant code/docs first.",
		"Keep scope to requested behavior.",
		"Update docs when behavior, commands, or structure change.",
		"Do not overwrite unrelated user changes.",
	}
}

func defaultStandards() []string {
	return []string{
		"Match style in touched files.",
		"Add abstractions only for real duplication or complexity.",
		"Use structured parsers when practical.",
	}
}

func defaultVerification() []string {
	return []string{
		"Run narrow checks first; broaden for shared behavior.",
		"Record reason when checks cannot run.",
		"Update tests for user-visible or shared behavior.",
	}
}

func itoa(value int) string {
	if value == 0 {
		return "0"
	}
	var digits []byte
	for value > 0 {
		digits = append(digits, byte('0'+value%10))
		value /= 10
	}
	for i, j := 0, len(digits)-1; i < j; i, j = i+1, j-1 {
		digits[i], digits[j] = digits[j], digits[i]
	}
	return string(digits)
}

func dedupe(items []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" || seen[item] {
			continue
		}
		seen[item] = true
		out = append(out, item)
	}
	return out
}

func dedupeCommands(commands []Command) []Command {
	seen := map[string]bool{}
	out := []Command{}
	for _, command := range commands {
		if command.Command == "" || seen[command.Command] {
			continue
		}
		seen[command.Command] = true
		out = append(out, command)
	}
	return out
}
