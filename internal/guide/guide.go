package guide

const (
	AgentsFile      = "AGENTS.md"
	ClaudeFile      = "CLAUDE.md"
	CustomSectionID = "squire-custom"
)

type Section struct {
	ID        string
	Title     string
	Required  bool
	Preserved bool
	Body      string
}

func AgentSections() []Section {
	return []Section{
		{
			ID:       "project-overview",
			Title:    "Project Overview",
			Required: true,
			Body:     "{{ bullets .Project.Overview }}",
		},
		{
			ID:       "technology-stack",
			Title:    "Technology Stack",
			Required: true,
			Body:     "{{ bullets .Project.Technologies }}",
		},
		{
			ID:       "project-structure",
			Title:    "Project Structure",
			Required: true,
			Body:     "{{ bullets .Project.Structure }}",
		},
		{
			ID:       "commands",
			Title:    "Commands",
			Required: true,
			Body:     "{{ commands .Project.Commands }}",
		},
		{
			ID:       "cli-tools",
			Title:    "CLI Tools for Agents",
			Required: true,
			Body:     "{{ tools .Tools }}",
		},
		{
			ID:       "environment",
			Title:    "Environment and Configuration",
			Required: true,
			Body:     "{{ bullets .Project.Environment }}",
		},
		{
			ID:       "development-workflow",
			Title:    "Development Workflow",
			Required: true,
			Body:     "{{ bullets .Project.Workflow }}",
		},
		{
			ID:       "coding-standards",
			Title:    "Coding Standards",
			Required: true,
			Body:     "{{ bullets .Project.Standards }}",
		},
		{
			ID:       "testing-and-verification",
			Title:    "Testing and Verification",
			Required: true,
			Body:     "{{ bullets .Project.Verification }}",
		},
		{
			ID:       "agent-operating-rules",
			Title:    "Agent Operating Rules",
			Required: true,
			Body: `- Treat this file as project map, not encyclopedia.
- Link source/docs for deep detail.
- Surface missing or conflicting instructions.
- Keep Squire comments for ` + "`squire analyze`" + `.`,
		},
		{
			ID:        CustomSectionID,
			Title:     "Squire Custom Notes",
			Required:  false,
			Preserved: true,
			Body: `- Add project-specific notes here.
- Squire preserves this section across regeneration.`,
		},
	}
}

func ClaudeSections() []Section {
	return []Section{
		{
			ID:        "claude-code",
			Title:     "Claude Code",
			Required:  true,
			Preserved: false,
			Body: `- Shared guidance: ` + "`AGENTS.md`" + `.
- Claude-only notes: this file.
- Personal/session memory: ` + "`/memory`" + `.`,
		},
	}
}

func ExpectedForTarget(target string) []Section {
	if target == "claude" {
		return ClaudeSections()
	}
	return AgentSections()
}
