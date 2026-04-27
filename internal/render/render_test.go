package render

import (
	"strings"
	"testing"

	"github.com/AndreBaltazar8/squire/internal/config"
	"github.com/AndreBaltazar8/squire/internal/guide"
	"github.com/AndreBaltazar8/squire/internal/project"
)

func TestClaudeImportsAgents(t *testing.T) {
	body, err := Claude(config.Config{}, project.Info{Name: "demo"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(body, "@AGENTS.md") {
		t.Fatalf("expected AGENTS import, got:\n%s", body)
	}
	if !strings.Contains(body, "## Claude Code") {
		t.Fatalf("expected claude section heading, got:\n%s", body)
	}
	if strings.Contains(body, "<!--") {
		t.Fatalf("expected markerless Claude output, got:\n%s", body)
	}
}

func TestAgentsIncludesDesignSystemSection(t *testing.T) {
	body, err := Agents(config.Config{}, project.Info{
		Name:   "demo",
		Design: []string{"`DESIGN.md` is the visual design system source of truth."},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(body, "## Design System") {
		t.Fatalf("expected design-system section heading, got:\n%s", body)
	}
	if !strings.Contains(body, "`DESIGN.md` is the visual design system source of truth.") {
		t.Fatalf("expected design guidance, got:\n%s", body)
	}
}

func TestAgentsSkipsEmptyDesignSystemSection(t *testing.T) {
	body, err := Agents(config.Config{}, project.Info{Name: "demo"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(body, "## Design System") {
		t.Fatalf("expected empty design-system section to be skipped, got:\n%s", body)
	}
}

func TestPreservedSectionOverridesDefault(t *testing.T) {
	body, err := Agents(config.Config{}, project.Info{Name: "demo"}, map[string]string{
		guide.CustomSectionID: "- Keep this note.",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(body, "- Keep this note.") {
		t.Fatalf("expected preserved note, got:\n%s", body)
	}
	if strings.Contains(body, "Add project-specific notes here") {
		t.Fatalf("expected default custom note to be replaced, got:\n%s", body)
	}

	preserved := PreservedSections([]byte(body))
	if preserved[guide.CustomSectionID] != "- Keep this note." {
		t.Fatalf("preserved = %#v", preserved)
	}
}

func TestToolsRenderStructured(t *testing.T) {
	body, err := Agents(config.Config{
		CLITools: []config.ToolConfig{{
			Name:        "playwright",
			Command:     "playwright",
			Description: "Rendered UI helper",
			When:        "Use for rendered UI verification",
			Examples:    []string{"playwright test", "playwright codegen"},
		}},
	}, project.Info{Name: "demo"}, nil)
	if err != nil {
		t.Fatal(err)
	}

	wants := []string{
		"- **`playwright`** (`playwright`) — Rendered UI helper.",
		"  - When: Use for rendered UI verification.",
		"  - Examples:",
		"    - `playwright test`",
		"    - `playwright codegen`",
	}
	for _, want := range wants {
		if !strings.Contains(body, want) {
			t.Fatalf("missing %q in:\n%s", want, body)
		}
	}
	// Make sure the old single-line semicolon shape is gone so nobody
	// accidentally regresses the format.
	if strings.Contains(body, "; examples:") {
		t.Fatalf("old single-line tool render leaked through:\n%s", body)
	}
}
