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

	got, err := resolveGenerationComponents(dir, []string{"svelte"}, false, []config.Component{{ID: "svelte"}}, config.ProjectConfig{})
	if err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Fatalf("components = %#v", got)
	}
}

func TestResolveGenerationComponentsUsesProjectConfig(t *testing.T) {
	dir := t.TempDir()
	components := []config.Component{
		{ID: "go", Detectors: config.ComponentDetectors{Any: []string{"go.mod"}}},
		{ID: "svelte"},
	}
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module demo\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := resolveGenerationComponents(dir, nil, false, components, config.ProjectConfig{Components: []string{"svelte"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0] != "svelte" {
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

func TestAnalyzePathReportsComponentsAndOutOfDate(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "AGENTS.md")
	if err := os.WriteFile(path, []byte(testAgentsBody(t, "Keep scope tight.")), 0o644); err != nil {
		t.Fatal(err)
	}

	report, err := analyzePath(path, config.Config{}, testProjectInfo("Keep scope focused."))
	if err != nil {
		t.Fatal(err)
	}
	if len(report.ExpectedComponents) != 1 || report.ExpectedComponents[0] != "go" {
		t.Fatalf("components = %#v", report.ExpectedComponents)
	}
	if !report.OutOfDate {
		t.Fatal("expected out-of-date report")
	}
}

func testAgentsBody(t *testing.T, workflow string) string {
	t.Helper()
	body, err := render.Agents(config.Config{}, testProjectInfo(workflow), nil)
	if err != nil {
		t.Fatal(err)
	}
	return body
}

func testProjectInfo(workflow string) project.Info {
	return project.Info{
		Name:         "demo",
		Components:   []string{"go"},
		Overview:     []string{"Demo project."},
		Technologies: []string{"Go."},
		Structure:    []string{"`cmd/` contains CLI entrypoints."},
		Commands:     []project.Command{{Command: "go test ./...", Description: "tests"}},
		Environment:  []string{"No secrets."},
		Workflow:     []string{workflow},
		Standards:    []string{"Match style."},
		Verification: []string{"Run tests."},
	}
}
