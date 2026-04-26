package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	agentpkg "squire/internal/agent"
	"squire/internal/analyze"
	"squire/internal/config"
	"squire/internal/guide"
	"squire/internal/project"
	"squire/internal/render"
)

type rootOptions struct {
	ConfigDir string
	CWD       string
}

func NewRootCommand() *cobra.Command {
	opts := &rootOptions{}

	root := &cobra.Command{
		Use:           "squire",
		Short:         "Normalize AI agent project instruction files",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	root.PersistentFlags().StringVar(&opts.ConfigDir, "config-dir", "", "config directory (default: ~/.config/squire or SQUIRE_CONFIG_DIR)")
	root.PersistentFlags().StringVar(&opts.CWD, "cwd", "", "project working directory (default: current directory)")

	root.AddCommand(newBrowseCommand(opts))
	root.AddCommand(newCLICommand(opts))
	root.AddCommand(newDetectCommand(opts))
	root.AddCommand(newDownloadCommand(opts))
	root.AddCommand(newAnalyzeCommand(opts))
	root.AddCommand(newGenerateCommand(opts))

	return root
}

func newCLICommand(opts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cli",
		Short: "Manage CLI tools included in generated agent guides",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCLIList(cmd, opts, false)
		},
	}

	var jsonOut bool
	listCmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List configured CLI tools and installation status",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCLIList(cmd, opts, jsonOut)
		},
	}
	listCmd.Flags().BoolVar(&jsonOut, "json", false, "print JSON")

	var description string
	var when string
	var examples []string
	addCmd := &cobra.Command{
		Use:   "add <name> [command]",
		Short: "Add or update a CLI tool",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.LoadConfig(opts.ConfigDir)
			if err != nil {
				return err
			}

			name := args[0]
			tool, index := findTool(cfg.CLITools, name)
			created := index == -1
			tool.Name = name
			if len(args) == 2 {
				tool.Command = args[1]
			}
			if tool.Command == "" {
				tool.Command = name
			}
			if cmd.Flags().Changed("description") {
				tool.Description = description
			}
			if cmd.Flags().Changed("when") {
				tool.When = when
			}
			if cmd.Flags().Changed("example") {
				tool.Examples = examples
			}

			if created {
				cfg.CLITools = append(cfg.CLITools, tool)
			} else {
				cfg.CLITools[index] = tool
			}
			if err := config.SaveConfig(opts.ConfigDir, cfg); err != nil {
				return err
			}

			status := "updated"
			if created {
				status = "added"
			}
			fmt.Fprintf(cmd.OutOrStdout(), "%s %s (`%s`)\n", status, tool.Name, tool.Command)
			return nil
		},
	}
	addCmd.Flags().StringVar(&description, "description", "", "tool description for generated guides")
	addCmd.Flags().StringVar(&when, "when", "", "when agents should use the tool")
	addCmd.Flags().StringArrayVar(&examples, "example", nil, "example command; may be repeated")

	removeCmd := &cobra.Command{
		Use:     "remove <name>",
		Aliases: []string{"rm"},
		Short:   "Remove a CLI tool from the global Squire config",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.LoadConfig(opts.ConfigDir)
			if err != nil {
				return err
			}
			_, index := findTool(cfg.CLITools, args[0])
			if index == -1 {
				return fmt.Errorf("CLI tool %q is not configured", args[0])
			}
			removed := cfg.CLITools[index]
			cfg.CLITools = append(cfg.CLITools[:index], cfg.CLITools[index+1:]...)
			if err := config.SaveConfig(opts.ConfigDir, cfg); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "removed %s\n", removed.Name)
			return nil
		},
	}

	cmd.AddCommand(listCmd, addCmd, removeCmd)
	return cmd
}

func newDetectCommand(opts *rootOptions) *cobra.Command {
	var jsonOut bool
	cmd := &cobra.Command{
		Use:   "detect",
		Short: "Detect installed agents and configured CLI tools",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.LoadConfig(opts.ConfigDir)
			if err != nil {
				return err
			}
			cwd, err := resolveCWD(opts.CWD)
			if err != nil {
				return err
			}
			components, err := config.LoadComponents(opts.ConfigDir)
			if err != nil {
				return err
			}
			projectInfo, err := project.Detect(cwd, "", nil, components)
			if err != nil {
				return err
			}
			out := struct {
				Agents []agentpkg.Detection     `json:"agents"`
				Tools  []agentpkg.ToolDetection `json:"tools"`
			}{
				Agents: agentpkg.DetectAgents(cfg.Agents),
				Tools:  agentpkg.DetectTools(config.MergeTools(cfg.CLITools, projectInfo.Tools)),
			}
			if jsonOut {
				return writeJSON(cmd, out)
			}

			fmt.Fprintln(cmd.OutOrStdout(), "Agents:")
			for _, item := range out.Agents {
				fmt.Fprintf(cmd.OutOrStdout(), "  %s\n", detectionLine(item.Name, item.Command, item.Installed, item.Path))
			}
			fmt.Fprintln(cmd.OutOrStdout(), "CLI tools:")
			for _, item := range out.Tools {
				fmt.Fprintf(cmd.OutOrStdout(), "  %s\n", detectionLine(item.Name, item.Command, item.Installed, item.Path))
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "print JSON")
	return cmd
}

func newAnalyzeCommand(opts *rootOptions) *cobra.Command {
	var jsonOut bool
	var strict bool
	cmd := &cobra.Command{
		Use:   "analyze [file]",
		Short: "Analyze Squire section coverage in AGENTS.md or CLAUDE.md",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := config.LoadConfig(opts.ConfigDir)
			if err != nil {
				return err
			}
			cwd, err := resolveCWD(opts.CWD)
			if err != nil {
				return err
			}

			paths := []string{}
			if len(args) == 1 {
				paths = append(paths, args[0])
			} else {
				for _, name := range []string{guide.AgentsFile, guide.ClaudeFile} {
					path := filepath.Join(cwd, name)
					if _, err := os.Stat(path); err == nil {
						paths = append(paths, path)
					}
				}
				if len(paths) == 0 {
					return fmt.Errorf("no %s or %s found in %s", guide.AgentsFile, guide.ClaudeFile, cwd)
				}
			}

			reports := make([]analyze.Report, 0, len(paths))
			for _, path := range paths {
				if !filepath.IsAbs(path) {
					path = filepath.Join(cwd, path)
				}
				report, err := analyzePath(path)
				if err != nil {
					return err
				}
				reports = append(reports, report)
			}

			if jsonOut {
				if len(reports) == 1 {
					return writeJSON(cmd, reports[0])
				}
				return writeJSON(cmd, reports)
			}

			for i, report := range reports {
				if i > 0 {
					fmt.Fprintln(cmd.OutOrStdout())
				}
				printReport(cmd, report)
			}
			if strict {
				for _, report := range reports {
					if !analyze.IsComplete(report) {
						return fmt.Errorf("%s is missing required Squire sections", report.File)
					}
				}
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "print JSON")
	cmd.Flags().BoolVar(&strict, "strict", false, "exit non-zero when required sections are missing")
	return cmd
}

func newGenerateCommand(opts *rootOptions) *cobra.Command {
	var force bool
	var stdout bool
	var check bool
	var projectName string
	var componentFlags []string
	var interactive bool

	cmd := &cobra.Command{
		Use:     "generate [agents|claude|all]",
		Aliases: []string{"normalize"},
		Short:   "Generate or normalize project instruction files",
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			target := "all"
			if len(args) == 1 {
				target = strings.ToLower(args[0])
			}
			if target != "agents" && target != "claude" && target != "all" {
				return fmt.Errorf("unknown generation target %q", target)
			}
			if stdout && check {
				return errors.New("--stdout and --check cannot be used together")
			}

			cfg, err := config.LoadConfig(opts.ConfigDir)
			if err != nil {
				return err
			}
			components, err := config.LoadComponents(opts.ConfigDir)
			if err != nil {
				return err
			}
			cwd, err := resolveCWD(opts.CWD)
			if err != nil {
				return err
			}
			componentExplicit := cmd.Flags().Changed("component")
			selectedComponents, err := resolveGenerationComponents(cwd, target, componentFlags, componentExplicit, components)
			if err != nil {
				return err
			}
			if interactive {
				selectedComponents, err = interactiveComponents(cmd, cwd, components, selectedComponents)
				if err != nil {
					return err
				}
			}
			projectInfo, err := project.Detect(cwd, projectName, selectedComponents, components)
			if err != nil {
				return err
			}

			files, err := renderTargets(target, cfg, projectInfo)
			if err != nil {
				return err
			}
			for i, file := range files {
				path := filepath.Join(cwd, file.Name)
				if file.Preserved == nil {
					preserved, err := readPreservedSections(path)
					if err != nil {
						return err
					}
					file.Preserved = preserved
					file.Body, err = renderTarget(file.Name, cfg, projectInfo, file.Preserved)
					if err != nil {
						return err
					}
				}
				if stdout {
					if len(files) > 1 {
						if i > 0 {
							fmt.Fprintln(cmd.OutOrStdout())
						}
						fmt.Fprintf(cmd.OutOrStdout(), "--- %s ---\n", file.Name)
					}
					fmt.Fprint(cmd.OutOrStdout(), file.Body)
					continue
				}

				status, err := writeGenerated(path, []byte(file.Body), force, check)
				if err != nil {
					return err
				}
				fmt.Fprintf(cmd.OutOrStdout(), "%s %s\n", status, path)
			}
			return nil
		},
	}
	cmd.Flags().BoolVarP(&force, "force", "f", false, "overwrite unmanaged existing files")
	cmd.Flags().BoolVar(&stdout, "stdout", false, "print generated content instead of writing files")
	cmd.Flags().BoolVar(&check, "check", false, "verify generated files are up to date without writing")
	cmd.Flags().StringVar(&projectName, "project-name", "", "override detected project name")
	cmd.Flags().StringArrayVar(&componentFlags, "component", nil, "component id; may be repeated or comma-separated")
	cmd.Flags().BoolVarP(&interactive, "interactive", "i", false, "select components interactively before generating")
	return cmd
}

type generatedFile struct {
	Name      string
	Body      string
	Preserved map[string]string
}

func renderTargets(target string, cfg config.Config, projectInfo project.Info) ([]generatedFile, error) {
	var files []generatedFile
	if target == "agents" || target == "all" {
		body, err := render.Agents(cfg, projectInfo, nil)
		if err != nil {
			return nil, err
		}
		files = append(files, generatedFile{Name: guide.AgentsFile, Body: body})
	}
	if target == "claude" || target == "all" {
		body, err := render.Claude(cfg, projectInfo, nil)
		if err != nil {
			return nil, err
		}
		files = append(files, generatedFile{Name: guide.ClaudeFile, Body: body})
	}
	return files, nil
}

func renderTarget(name string, cfg config.Config, projectInfo project.Info, preserved map[string]string) (string, error) {
	switch name {
	case guide.AgentsFile:
		return render.Agents(cfg, projectInfo, preserved)
	case guide.ClaudeFile:
		return render.Claude(cfg, projectInfo, preserved)
	default:
		return "", fmt.Errorf("unknown generated file %q", name)
	}
}

func readPreservedSections(path string) (map[string]string, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	return render.PreservedSections(body), nil
}

func resolveGenerationComponents(cwd, target string, requested []string, explicit bool, components []config.Component) ([]string, error) {
	if explicit {
		ids := parseComponentArgs(requested)
		ids = mergeComponentIDs(project.DetectedComponentIDs(cwd, components), ids)
		if err := validateComponentIDs(ids, components); err != nil {
			return nil, err
		}
		return ids, nil
	}

	for _, filename := range componentLookupFiles(target) {
		body, err := os.ReadFile(filepath.Join(cwd, filename))
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return nil, err
		}
		if ids := render.ManagedComponents(body); len(ids) > 0 {
			return ids, nil
		}
	}
	return nil, nil
}

func componentLookupFiles(target string) []string {
	switch target {
	case "agents":
		return []string{guide.AgentsFile, guide.ClaudeFile}
	case "claude":
		return []string{guide.ClaudeFile, guide.AgentsFile}
	default:
		return []string{guide.AgentsFile, guide.ClaudeFile}
	}
}

func parseComponentArgs(values []string) []string {
	var ids []string
	for _, value := range values {
		for _, part := range strings.Split(value, ",") {
			part = strings.TrimSpace(part)
			if part != "" {
				ids = append(ids, part)
			}
		}
	}
	return mergeComponentIDs(ids)
}

func mergeComponentIDs(groups ...[]string) []string {
	seen := map[string]bool{}
	var out []string
	for _, group := range groups {
		for _, id := range group {
			id = strings.TrimSpace(id)
			if id == "" || seen[id] {
				continue
			}
			seen[id] = true
			out = append(out, id)
		}
	}
	return out
}

func validateComponentIDs(ids []string, components []config.Component) error {
	if len(ids) == 0 {
		return nil
	}
	known := map[string]bool{}
	for _, component := range components {
		if component.ID != "" {
			known[component.ID] = true
		}
	}
	var unknown []string
	for _, id := range ids {
		if !known[id] {
			unknown = append(unknown, id)
		}
	}
	if len(unknown) > 0 {
		return fmt.Errorf("unknown component %s; available components: %s", strings.Join(unknown, ", "), availableComponents(components))
	}
	return nil
}

func availableComponents(components []config.Component) string {
	var ids []string
	for _, component := range components {
		if component.ID != "" {
			ids = append(ids, component.ID)
		}
	}
	if len(ids) == 0 {
		return "none"
	}
	return strings.Join(ids, ", ")
}

func writeGenerated(path string, body []byte, force, check bool) (string, error) {
	existing, err := os.ReadFile(path)
	if err == nil {
		if string(existing) == string(body) {
			if check {
				return "up-to-date", nil
			}
			return "kept", nil
		}
		if check {
			return "", fmt.Errorf("%s is out of date", path)
		}
		if !force && !render.IsManaged(existing) {
			return "", fmt.Errorf("%s exists and is not Squire-managed; rerun with --force to overwrite", path)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", err
	} else if check {
		return "", fmt.Errorf("%s does not exist", path)
	}

	if err := os.WriteFile(path, body, 0o644); err != nil {
		return "", err
	}
	if err == nil {
		return "updated", nil
	}
	return "wrote", nil
}

func analyzePath(path string) (analyze.Report, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return analyze.Report{}, err
	}
	target := "agents"
	if strings.EqualFold(filepath.Base(path), guide.ClaudeFile) {
		target = "claude"
	}
	expected := analyze.ExpectedForTarget(target)
	return analyze.File(string(body), path, expected), nil
}

func printReport(cmd *cobra.Command, report analyze.Report) {
	fmt.Fprintf(cmd.OutOrStdout(), "%s\n", report.File)
	fmt.Fprintf(cmd.OutOrStdout(), "  managed: %t\n", report.HasManagedMarker)
	fmt.Fprintf(cmd.OutOrStdout(), "  present: %s\n", joinOrNone(report.Present))
	fmt.Fprintf(cmd.OutOrStdout(), "  missing: %s\n", joinOrNone(report.Missing))
	if len(report.UntaggedMatches) > 0 {
		values := make([]string, 0, len(report.UntaggedMatches))
		for _, item := range report.UntaggedMatches {
			values = append(values, item.SectionID+" ("+item.Heading+")")
		}
		fmt.Fprintf(cmd.OutOrStdout(), "  untagged matching headings: %s\n", strings.Join(values, ", "))
	}
	if len(report.UnknownTagged) > 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "  unknown tagged sections: %s\n", strings.Join(report.UnknownTagged, ", "))
	}
}

func runCLIList(cmd *cobra.Command, opts *rootOptions, jsonOut bool) error {
	cfg, err := config.LoadConfig(opts.ConfigDir)
	if err != nil {
		return err
	}
	detected := agentpkg.DetectTools(cfg.CLITools)
	if jsonOut {
		return writeJSON(cmd, detected)
	}
	for _, item := range detected {
		fmt.Fprintf(cmd.OutOrStdout(), "%s\n", detectionLine(item.Name, item.Command, item.Installed, item.Path))
	}
	return nil
}

func findTool(tools []config.ToolConfig, name string) (config.ToolConfig, int) {
	for i, tool := range tools {
		if strings.EqualFold(tool.Name, name) {
			return tool, i
		}
	}
	return config.ToolConfig{}, -1
}

func resolveCWD(value string) (string, error) {
	if value == "" {
		return os.Getwd()
	}
	return filepath.Abs(value)
}

func writeJSON(cmd *cobra.Command, value any) error {
	encoder := json.NewEncoder(cmd.OutOrStdout())
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}

func detectionLine(name, command string, installed bool, path string) string {
	state := "missing"
	if installed {
		state = "installed"
	}
	if path != "" {
		return fmt.Sprintf("%s (`%s`): %s at %s", name, command, state, path)
	}
	return fmt.Sprintf("%s (`%s`): %s", name, command, state)
}

func joinOrNone(values []string) string {
	if len(values) == 0 {
		return "none"
	}
	return strings.Join(values, ", ")
}
