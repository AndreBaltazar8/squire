package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"squire/internal/config"
	"squire/internal/project"
	"squire/internal/render"
)

func TestResolveGenerationComponentsIsStatelessWithoutExplicitSelection(t *testing.T) {
	dir := t.TempDir()

	got, err := resolveGenerationComponents(dir, "all", []string{"svelte"}, false, []config.Component{{ID: "svelte"}})
	if err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Fatalf("components = %#v", got)
	}
}

func TestWriteGeneratedAllowsSquireHeadingFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "AGENTS.md")
	oldBody := testAgentsBody(t, "Keep scope tight.")
	newBody := strings.Replace(oldBody, "Keep scope tight.", "Keep scope focused.", 1)
	if err := os.WriteFile(path, []byte(oldBody), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := writeGenerated(path, []byte(newBody), false, false); err != nil {
		t.Fatal(err)
	}
}

func TestWriteGeneratedRejectsUnrecognizedMarkerlessFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "AGENTS.md")
	if err := os.WriteFile(path, []byte("# Notes\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := writeGenerated(path, []byte("# New\n"), false, false); err == nil {
		t.Fatal("expected unmanaged markerless file to require force")
	}
}

func testAgentsBody(t *testing.T, workflow string) string {
	t.Helper()
	body, err := render.Agents(config.Config{}, project.Info{
		Name:         "demo",
		Overview:     []string{"Demo project."},
		Technologies: []string{"Go."},
		Structure:    []string{"`cmd/` contains CLI entrypoints."},
		Commands:     []project.Command{{Command: "go test ./...", Description: "tests"}},
		Environment:  []string{"No secrets."},
		Workflow:     []string{workflow},
		Standards:    []string{"Match style."},
		Verification: []string{"Run tests."},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	return body
}
