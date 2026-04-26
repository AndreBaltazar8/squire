package render

import (
	"sort"
	"strings"
	"text/template"

	"squire/internal/config"
	"squire/internal/guide"
	"squire/internal/project"
)

type Context struct {
	Config    config.Config
	Project   project.Info
	Tools     []config.ToolConfig
	Preserved map[string]string
}

func Agents(cfg config.Config, projectInfo project.Info, preserved map[string]string) (string, error) {
	ctx := newContext(cfg, projectInfo, preserved)
	var out strings.Builder
	out.WriteString("# " + projectInfo.Name + " Agent Guide\n\n")

	for _, section := range guide.AgentSections() {
		rendered, err := renderSection(section, ctx)
		if err != nil {
			return "", err
		}
		if rendered == "" {
			continue
		}
		out.WriteString(rendered)
		out.WriteString("\n")
	}

	return ensureTrailingNewline(out.String()), nil
}

func Claude(cfg config.Config, projectInfo project.Info, preserved map[string]string) (string, error) {
	ctx := newContext(cfg, projectInfo, preserved)
	var out strings.Builder
	out.WriteString("@AGENTS.md\n\n")
	out.WriteString("# " + projectInfo.Name + " Claude Code Guide\n\n")
	out.WriteString("Shared guidance: `AGENTS.md`. Claude-specific notes below.\n\n")

	for _, section := range guide.ClaudeSections() {
		rendered, err := renderSection(section, ctx)
		if err != nil {
			return "", err
		}
		if rendered == "" {
			continue
		}
		out.WriteString(rendered)
		out.WriteString("\n")
	}

	return ensureTrailingNewline(out.String()), nil
}

func PreservedSections(body []byte) map[string]string {
	out := map[string]string{}
	for _, section := range guide.AgentSections() {
		if !section.Preserved {
			continue
		}
		if body := sectionBody(string(body), section); body != "" {
			out[section.ID] = body
		}
	}
	for _, section := range guide.ClaudeSections() {
		if !section.Preserved {
			continue
		}
		if body := sectionBody(string(body), section); body != "" {
			out[section.ID] = body
		}
	}
	return out
}

func newContext(cfg config.Config, projectInfo project.Info, preserved map[string]string) Context {
	return Context{
		Config:    cfg,
		Project:   projectInfo,
		Tools:     config.MergeTools(cfg.CLITools, projectInfo.Tools),
		Preserved: preserved,
	}
}

func renderSection(section guide.Section, ctx Context) (string, error) {
	var body string
	if section.Preserved {
		body = strings.TrimSpace(ctx.Preserved[section.ID])
	}
	if body == "" {
		var err error
		body, err = executeTemplate(section.ID, section.Body, ctx)
		if err != nil {
			return "", err
		}
	}

	body = strings.TrimSpace(body)
	if body == "" && section.SkipEmpty {
		return "", nil
	}
	if body == "" {
		body = "- No guidance configured yet."
	}

	var out strings.Builder
	out.WriteString("## " + section.Title + "\n\n")
	out.WriteString(body)
	out.WriteString("\n")
	return out.String(), nil
}

func sectionBody(content string, section guide.Section) string {
	return headingSectionBody(content, section.Title)
}

func headingSectionBody(content, title string) string {
	expectedSlug := slug(title)
	lines := strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n")
	start := -1
	startLevel := 0

	for i, line := range lines {
		level, heading, ok := markdownHeading(line)
		if !ok || slug(heading) != expectedSlug {
			continue
		}
		start = i + 1
		startLevel = level
		break
	}
	if start == -1 {
		return ""
	}

	end := len(lines)
	for i := start; i < len(lines); i++ {
		level, _, ok := markdownHeading(lines[i])
		if ok && level <= startLevel {
			end = i
			break
		}
	}

	return strings.TrimSpace(strings.Join(lines[start:end], "\n"))
}

func markdownHeading(line string) (int, string, bool) {
	line = strings.TrimSpace(line)
	level := 0
	for level < len(line) && level < 6 && line[level] == '#' {
		level++
	}
	if level == 0 || level >= len(line) || line[level] != ' ' {
		return 0, "", false
	}
	title := strings.TrimSpace(line[level+1:])
	title = strings.TrimSpace(strings.TrimRight(title, "#"))
	if title == "" {
		return 0, "", false
	}
	return level, title, true
}

func slug(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var out strings.Builder
	lastDash := false
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			out.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			out.WriteRune('-')
			lastDash = true
		}
	}
	return strings.Trim(out.String(), "-")
}

func executeTemplate(name, body string, ctx Context) (string, error) {
	tpl, err := template.New(name).Funcs(template.FuncMap{
		"bullets":         bullets,
		"optionalBullets": optionalBullets,
		"commands":        commands,
		"tools":           tools,
	}).Parse(body)
	if err != nil {
		return "", err
	}

	var out strings.Builder
	if err := tpl.Execute(&out, ctx); err != nil {
		return "", err
	}
	return out.String(), nil
}

func optionalBullets(items []string) string {
	if len(items) == 0 {
		return ""
	}
	return bullets(items)
}

func bullets(items []string) string {
	if len(items) == 0 {
		return "- No guidance configured yet."
	}
	var out strings.Builder
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		out.WriteString("- " + item + "\n")
	}
	return strings.TrimRight(out.String(), "\n")
}

func commands(items []project.Command) string {
	if len(items) == 0 {
		return "- Add setup, development, test, lint, and build commands for this project."
	}
	var out strings.Builder
	for _, item := range items {
		command := strings.TrimSpace(item.Command)
		if command == "" {
			continue
		}
		description := strings.TrimSpace(item.Description)
		if description == "" {
			out.WriteString("- `" + command + "`\n")
			continue
		}
		out.WriteString("- `" + command + "`: " + strings.TrimSuffix(description, ".") + "\n")
	}
	return strings.TrimRight(out.String(), "\n")
}

func tools(items []config.ToolConfig) string {
	if len(items) == 0 {
		return "- Add project-specific CLI tools with `squire cli add`."
	}

	sorted := append([]config.ToolConfig(nil), items...)
	sort.SliceStable(sorted, func(i, j int) bool {
		return sorted[i].Name < sorted[j].Name
	})

	var out strings.Builder
	for _, item := range sorted {
		name := strings.TrimSpace(item.Name)
		command := strings.TrimSpace(item.Command)
		if name == "" {
			continue
		}
		if command == "" {
			command = name
		}
		line := "- `" + name + "` (`" + command + "`)"
		if item.Description != "" {
			line += ": " + strings.TrimSuffix(item.Description, ".")
		}
		if item.When != "" {
			line += "; " + strings.TrimSuffix(item.When, ".")
		}
		if len(item.Examples) > 0 {
			examples := make([]string, 0, len(item.Examples))
			for _, example := range item.Examples {
				example = strings.TrimSpace(example)
				if example != "" {
					examples = append(examples, "`"+example+"`")
				}
			}
			if len(examples) > 0 {
				line += "; examples: " + strings.Join(examples, ", ")
			}
		}
		out.WriteString(line + "\n")
	}
	return strings.TrimRight(out.String(), "\n")
}

func ensureTrailingNewline(value string) string {
	return strings.TrimRight(value, "\n") + "\n"
}
