package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"squire/internal/config"
)

func newComponentCommand(opts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "component",
		Aliases: []string{"components"},
		Short:   "Manage installed local components",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runComponentList(cmd, opts)
		},
	}

	listCmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List installed components",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runComponentList(cmd, opts)
		},
	}

	removeCmd := &cobra.Command{
		Use:     "remove <id>",
		Aliases: []string{"rm"},
		Short:   "Remove an installed component",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			configDir, err := resolvedConfigDir(opts.ConfigDir)
			if err != nil {
				return err
			}
			if err := config.EnsureDefaults(configDir); err != nil {
				return err
			}
			path, err := componentPath(configDir, args[0])
			if err != nil {
				return err
			}
			if err := os.Remove(path); err != nil {
				if os.IsNotExist(err) {
					return fmt.Errorf("component %q is not installed", args[0])
				}
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "removed %s\n", path)
			return nil
		},
	}

	cmd.AddCommand(listCmd, removeCmd)
	return cmd
}

func runComponentList(cmd *cobra.Command, opts *rootOptions) error {
	configDir, err := resolvedConfigDir(opts.ConfigDir)
	if err != nil {
		return err
	}
	if err := config.EnsureDefaults(configDir); err != nil {
		return err
	}
	components, err := config.LoadComponents(configDir)
	if err != nil {
		return err
	}
	if len(components) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No installed components.")
		return nil
	}
	for _, component := range components {
		line := component.ID
		if component.Description != "" {
			line += ": " + strings.TrimSuffix(component.Description, ".")
		}
		fmt.Fprintln(cmd.OutOrStdout(), line)
	}
	return nil
}

func componentPath(configDir, id string) (string, error) {
	id = strings.TrimSpace(id)
	if id == "" || strings.ContainsAny(id, `/\`) {
		return "", fmt.Errorf("invalid component id %q", id)
	}
	return filepath.Join(configDir, config.ComponentsDir, id+".yaml"), nil
}

func resolvedConfigDir(value string) (string, error) {
	if value != "" {
		return filepath.Abs(value)
	}
	return config.DefaultConfigDir()
}
