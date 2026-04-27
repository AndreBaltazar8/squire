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
