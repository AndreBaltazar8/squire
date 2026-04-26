package render

import (
	"strings"
	"testing"

	"squire/internal/config"
	"squire/internal/guide"
	"squire/internal/project"
)

func TestClaudeImportsAgents(t *testing.T) {
	body, err := Claude(config.Config{}, project.Info{Name: "demo"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(body, "@AGENTS.md") {
		t.Fatalf("expected AGENTS import, got:\n%s", body)
	}
	if !strings.Contains(body, "squire:start id=claude-code") {
		t.Fatalf("expected claude section marker, got:\n%s", body)
	}
}

func TestManagedComponents(t *testing.T) {
	body, err := Agents(config.Config{}, project.Info{Name: "demo", Components: []string{"svelte", "go"}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	got := ManagedComponents([]byte(body))
	if len(got) != 2 || got[0] != "go" || got[1] != "svelte" {
		t.Fatalf("ManagedComponents = %#v", got)
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
