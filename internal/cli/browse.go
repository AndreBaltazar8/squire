package cli

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func newBrowseCommand(opts *rootOptions) *cobra.Command {
	var jsonOut bool
	cmd := &cobra.Command{
		Use:   "browse [github-source[#component]]",
		Short: "Browse available Squire components from provider repos",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			raw := defaultProviderSource
			if len(args) == 1 {
				raw = args[0]
			}
			source, err := parseComponentSource(raw)
			if err != nil {
				return err
			}
			providers, err := browseProviders(source)
			if err != nil {
				return err
			}
			if jsonOut {
				return writeBrowseJSON(cmd, providers)
			}
			return printBrowse(cmd, providers, source.Selector)
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "print JSON")
	return cmd
}

type browseJSONProvider struct {
	ID            string                `json:"id"`
	Name          string                `json:"name"`
	Description   string                `json:"description,omitempty"`
	Source        string                `json:"source"`
	ComponentsDir string                `json:"components_dir"`
	Components    []browseJSONComponent `json:"components"`
}

type browseJSONComponent struct {
	ID          string `json:"id"`
	Description string `json:"description,omitempty"`
	Path        string `json:"path"`
}

func writeBrowseJSON(cmd *cobra.Command, providers []remoteProvider) error {
	out := make([]browseJSONProvider, 0, len(providers))
	for _, provider := range providers {
		item := browseJSONProvider{
			ID:            provider.ID,
			Name:          provider.Name,
			Description:   provider.Description,
			Source:        provider.SourceString,
			ComponentsDir: provider.ComponentsDir,
		}
		for _, component := range provider.Components {
			item.Components = append(item.Components, browseJSONComponent{
				ID:          component.ID,
				Description: component.Description,
				Path:        component.Path,
			})
		}
		out = append(out, item)
	}
	encoder := json.NewEncoder(cmd.OutOrStdout())
	encoder.SetIndent("", "  ")
	return encoder.Encode(out)
}

func printBrowse(cmd *cobra.Command, providers []remoteProvider, selector string) error {
	if len(providers) == 0 {
		if selector != "" {
			fmt.Fprintf(cmd.OutOrStdout(), "No component %q found.\n", selector)
			return nil
		}
		fmt.Fprintln(cmd.OutOrStdout(), "No component providers found.")
		return nil
	}
	for i, provider := range providers {
		if i > 0 {
			fmt.Fprintln(cmd.OutOrStdout())
		}
		header := provider.Name
		if strings.TrimSpace(header) == "" {
			header = provider.ID
		}
		if provider.SourceString != "" {
			header += " (" + provider.SourceString + ")"
		}
		fmt.Fprintln(cmd.OutOrStdout(), header)
		if provider.Description != "" {
			fmt.Fprintf(cmd.OutOrStdout(), "  %s\n", strings.TrimSuffix(provider.Description, "."))
		}
		if provider.ComponentsDir != "" {
			fmt.Fprintf(cmd.OutOrStdout(), "  components: %s\n", provider.ComponentsDir)
		}
		if len(provider.Components) == 0 {
			fmt.Fprintln(cmd.OutOrStdout(), "  no components found")
			continue
		}
		for _, component := range provider.Components {
			line := "  - " + component.ID
			if component.Description != "" {
				line += ": " + strings.TrimSuffix(component.Description, ".")
			}
			fmt.Fprintln(cmd.OutOrStdout(), line)
		}
	}
	return nil
}
