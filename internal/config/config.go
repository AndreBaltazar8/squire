package config

import (
	"errors"
	"os"
	"path/filepath"
	"sort"

	"gopkg.in/yaml.v3"
)

const (
	ConfigFileName        = "config.yaml"
	ComponentsDir         = "components"
	ProjectConfigFileName = "squire.yaml"
)

type Config struct {
	Version  int           `yaml:"version" json:"version"`
	Agents   []AgentConfig `yaml:"agents" json:"agents"`
	CLITools []ToolConfig  `yaml:"cli_tools" json:"cli_tools"`
}

type ProjectConfig struct {
	Components []string `yaml:"components" json:"components"`
	// Description, when set, overrides the auto-detected Project Overview
	// paragraph. Pin it in `squire.yaml` when the README's first paragraph
	// is not actually a description of the project (legacy READMEs that
	// open with a setup section, "How to get started" prefixes, meta
	// disclaimers about the README itself, etc.).
	Description string `yaml:"description,omitempty" json:"description,omitempty"`
}

type AgentConfig struct {
	Name        string   `yaml:"name" json:"name"`
	Command     string   `yaml:"command" json:"command"`
	Description string   `yaml:"description" json:"description"`
	Files       []string `yaml:"files" json:"files"`
}

type ToolConfig struct {
	Name        string   `yaml:"name" json:"name"`
	Command     string   `yaml:"command" json:"command"`
	Description string   `yaml:"description" json:"description"`
	When        string   `yaml:"when" json:"when"`
	Examples    []string `yaml:"examples" json:"examples"`
}

type Component struct {
	Version     int                `yaml:"version" json:"version"`
	ID          string             `yaml:"id" json:"id"`
	Description string             `yaml:"description" json:"description"`
	Detectors   ComponentDetectors `yaml:"detectors" json:"detectors"`
	Guidance    ComponentGuidance  `yaml:"guidance" json:"guidance"`
	CLITools    []ToolConfig       `yaml:"cli_tools" json:"cli_tools"`
}

type ComponentDetectors struct {
	Any []DetectorRule `yaml:"any" json:"any"`
	All []DetectorRule `yaml:"all" json:"all"`
}

// DetectorRule is a single match condition. It accepts two YAML forms:
//
//	- some/path/glob              # bare string, treated as Glob
//	- {file: name, contains: txt} # content-aware match
//
// A rule with only Glob (or only File) checks file existence. A rule with
// Contains additionally requires the matched file to contain the substring
// (case-insensitive). Regex, when set, is applied case-insensitively after
// any Contains check.
type DetectorRule struct {
	Glob     string `yaml:"glob,omitempty" json:"glob,omitempty"`
	File     string `yaml:"file,omitempty" json:"file,omitempty"`
	Contains string `yaml:"contains,omitempty" json:"contains,omitempty"`
	Regex    string `yaml:"regex,omitempty" json:"regex,omitempty"`
}

func (r *DetectorRule) UnmarshalYAML(node *yaml.Node) error {
	if node.Kind == yaml.ScalarNode {
		r.Glob = node.Value
		return nil
	}
	type alias DetectorRule
	var raw alias
	if err := node.Decode(&raw); err != nil {
		return err
	}
	*r = DetectorRule(raw)
	return nil
}

// Pattern returns the file/glob pattern this rule operates on.
func (r DetectorRule) Pattern() string {
	if r.File != "" {
		return r.File
	}
	return r.Glob
}

// NeedsContent reports whether the rule requires reading a file's content.
func (r DetectorRule) NeedsContent() bool {
	return r.Contains != "" || r.Regex != ""
}

type ComponentGuidance struct {
	Overview     []string           `yaml:"overview" json:"overview"`
	Technologies []string           `yaml:"technologies" json:"technologies"`
	Structure    []string           `yaml:"structure" json:"structure"`
	Design       []string           `yaml:"design" json:"design"`
	Commands     []ComponentCommand `yaml:"commands" json:"commands"`
	Environment  []string           `yaml:"environment" json:"environment"`
	Workflow     []string           `yaml:"workflow" json:"workflow"`
	Standards    []string           `yaml:"standards" json:"standards"`
	Verification []string           `yaml:"verification" json:"verification"`
}

type ComponentCommand struct {
	Command     string `yaml:"command" json:"command"`
	Description string `yaml:"description" json:"description"`
}

func DefaultConfigDir() (string, error) {
	if env := os.Getenv("SQUIRE_CONFIG_DIR"); env != "" {
		return filepath.Abs(env)
	}

	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "squire"), nil
}

func LoadConfig(configDir string) (Config, error) {
	if configDir == "" {
		var err error
		configDir, err = DefaultConfigDir()
		if err != nil {
			return Config{}, err
		}
	}

	if err := EnsureDefaults(configDir); err != nil {
		return Config{}, err
	}

	path := filepath.Join(configDir, ConfigFileName)
	raw, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return parseConfig(DefaultConfigYAML)
		}
		return Config{}, err
	}
	return parseConfig(string(raw))
}

func SaveConfig(configDir string, cfg Config) error {
	if configDir == "" {
		var err error
		configDir, err = DefaultConfigDir()
		if err != nil {
			return err
		}
	}
	if cfg.Version == 0 {
		cfg.Version = 1
	}
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		return err
	}
	raw, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	path := filepath.Join(configDir, ConfigFileName)
	tmp, err := os.CreateTemp(configDir, ConfigFileName+".*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)

	if _, err := tmp.Write(raw); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Chmod(0o644); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}

func LoadProjectConfig(root string) (ProjectConfig, error) {
	path := filepath.Join(root, ProjectConfigFileName)
	raw, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return ProjectConfig{}, nil
		}
		return ProjectConfig{}, err
	}

	var cfg ProjectConfig
	if err := yaml.Unmarshal(raw, &cfg); err != nil {
		return ProjectConfig{}, err
	}
	return cfg, nil
}

func LoadComponents(configDir string) ([]Component, error) {
	if configDir == "" {
		var err error
		configDir, err = DefaultConfigDir()
		if err != nil {
			return nil, err
		}
	}

	dir := filepath.Join(configDir, ComponentsDir)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}

	var components []Component
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".yaml" {
			continue
		}
		body, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			return nil, err
		}
		var component Component
		if err := yaml.Unmarshal(body, &component); err != nil {
			return nil, err
		}
		if component.ID == "" {
			component.ID = entry.Name()[:len(entry.Name())-len(".yaml")]
		}
		if component.Version == 0 {
			component.Version = 1
		}
		components = append(components, component)
	}
	sort.SliceStable(components, func(i, j int) bool {
		return components[i].ID < components[j].ID
	})
	return components, nil
}

func EnsureDefaults(configDir string) error {
	if configDir == "" {
		var err error
		configDir, err = DefaultConfigDir()
		if err != nil {
			return err
		}
	}

	if err := os.MkdirAll(configDir, 0o755); err != nil {
		return err
	}

	configPath := filepath.Join(configDir, ConfigFileName)
	if _, err := os.Stat(configPath); errors.Is(err, os.ErrNotExist) {
		if err := os.WriteFile(configPath, []byte(DefaultConfigYAML), 0o644); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	componentsDir := filepath.Join(configDir, ComponentsDir)
	if err := os.MkdirAll(componentsDir, 0o755); err != nil {
		return err
	}
	// Seed any default component that is not already present. Existing files
	// are left alone so user edits are preserved across upgrades, while new
	// defaults shipped in later versions still land on disk.
	for _, component := range DefaultComponents {
		path := filepath.Join(componentsDir, component.Name)
		if _, err := os.Stat(path); err == nil {
			continue
		} else if !errors.Is(err, os.ErrNotExist) {
			return err
		}
		if err := os.WriteFile(path, []byte(component.Body), 0o644); err != nil {
			return err
		}
	}
	return nil
}

func parseConfig(raw string) (Config, error) {
	var cfg Config
	if err := yaml.Unmarshal([]byte(raw), &cfg); err != nil {
		return Config{}, err
	}
	if cfg.Version == 0 {
		cfg.Version = 1
	}
	return cfg, nil
}

func MergeTools(base, overlay []ToolConfig) []ToolConfig {
	byName := map[string]ToolConfig{}
	order := []string{}
	add := func(tool ToolConfig) {
		if tool.Name == "" {
			return
		}
		if _, exists := byName[tool.Name]; !exists {
			order = append(order, tool.Name)
		}
		byName[tool.Name] = tool
	}

	for _, tool := range base {
		add(tool)
	}
	for _, tool := range overlay {
		add(tool)
	}

	out := make([]ToolConfig, 0, len(order))
	for _, name := range order {
		out = append(out, byName[name])
	}
	return out
}
