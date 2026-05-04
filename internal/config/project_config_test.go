package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadProjectConfigMissingFile(t *testing.T) {
	cfg, err := LoadProjectConfig(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Components) != 0 {
		t.Fatalf("components = %#v", cfg.Components)
	}
}

func TestLoadProjectConfigComponents(t *testing.T) {
	dir := t.TempDir()
	body := []byte("components:\n  - go\n  - svelte\n")
	if err := os.WriteFile(filepath.Join(dir, ProjectConfigFileName), body, 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadProjectConfig(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Components) != 2 || cfg.Components[0] != "go" || cfg.Components[1] != "svelte" {
		t.Fatalf("components = %#v", cfg.Components)
	}
}

func TestLoadProjectConfigCLITools(t *testing.T) {
	dir := t.TempDir()
	body := []byte("cli_tools:\n  - name: harness\n    command: game-krunker-harness\n    description: local-only harness\n    when: drive the harness\n    examples:\n      - harness cmd help\n")
	if err := os.WriteFile(filepath.Join(dir, ProjectConfigFileName), body, 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadProjectConfig(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.CLITools) != 1 {
		t.Fatalf("cli_tools = %#v", cfg.CLITools)
	}
	tool := cfg.CLITools[0]
	if tool.Name != "harness" || tool.Command != "game-krunker-harness" {
		t.Fatalf("tool = %#v", tool)
	}
	if tool.Description != "local-only harness" || tool.When != "drive the harness" {
		t.Fatalf("tool metadata = %#v", tool)
	}
	if len(tool.Examples) != 1 || tool.Examples[0] != "harness cmd help" {
		t.Fatalf("examples = %#v", tool.Examples)
	}
}

func TestSaveProjectConfigRoundTrip(t *testing.T) {
	dir := t.TempDir()
	original := ProjectConfig{
		Components: []string{"go"},
		CLITools: []ToolConfig{{
			Name:        "harness",
			Command:     "game-krunker-harness",
			Description: "local harness",
			When:        "use locally",
			Examples:    []string{"harness cmd help"},
		}},
	}
	if err := SaveProjectConfig(dir, original); err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadProjectConfig(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded.Components) != 1 || loaded.Components[0] != "go" {
		t.Fatalf("components = %#v", loaded.Components)
	}
	if len(loaded.CLITools) != 1 {
		t.Fatalf("cli_tools = %#v", loaded.CLITools)
	}
	tool := loaded.CLITools[0]
	if tool.Name != "harness" || tool.Command != "game-krunker-harness" {
		t.Fatalf("tool = %#v", tool)
	}
	if len(tool.Examples) != 1 || tool.Examples[0] != "harness cmd help" {
		t.Fatalf("examples = %#v", tool.Examples)
	}
}
