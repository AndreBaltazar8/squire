package agent

import (
	"os/exec"

	"squire/internal/config"
)

type Detection struct {
	Name        string   `json:"name"`
	Command     string   `json:"command"`
	Description string   `json:"description"`
	Files       []string `json:"files"`
	Installed   bool     `json:"installed"`
	Path        string   `json:"path,omitempty"`
}

type ToolDetection struct {
	Name        string   `json:"name"`
	Command     string   `json:"command"`
	Description string   `json:"description"`
	When        string   `json:"when"`
	Examples    []string `json:"examples"`
	Installed   bool     `json:"installed"`
	Path        string   `json:"path,omitempty"`
}

func DetectAgents(agents []config.AgentConfig) []Detection {
	out := make([]Detection, 0, len(agents))
	for _, item := range agents {
		command := item.Command
		if command == "" {
			command = item.Name
		}
		path, err := exec.LookPath(command)
		out = append(out, Detection{
			Name:        item.Name,
			Command:     command,
			Description: item.Description,
			Files:       append([]string(nil), item.Files...),
			Installed:   err == nil,
			Path:        path,
		})
	}
	return out
}

func DetectTools(tools []config.ToolConfig) []ToolDetection {
	out := make([]ToolDetection, 0, len(tools))
	for _, item := range tools {
		command := item.Command
		if command == "" {
			command = item.Name
		}
		path, err := exec.LookPath(command)
		out = append(out, ToolDetection{
			Name:        item.Name,
			Command:     command,
			Description: item.Description,
			When:        item.When,
			Examples:    append([]string(nil), item.Examples...),
			Installed:   err == nil,
			Path:        path,
		})
	}
	return out
}
