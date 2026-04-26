package cli

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"
)

func newBrowseCommand(opts *rootOptions) *cobra.Command {
	var jsonOut bool
	var interactive bool
	var force bool
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
			if interactive && jsonOut {
				return errors.New("--interactive and --json cannot be used together")
			}
			if jsonOut {
				return writeBrowseJSON(cmd, providers)
			}
			if interactive {
				selections, err := runBrowsePicker(cmd.InOrStdin(), cmd.ErrOrStderr(), providers)
				if err != nil {
					return err
				}
				if len(selections) == 0 {
					fmt.Fprintln(cmd.OutOrStdout(), "No components selected.")
					return nil
				}
				for _, selection := range selections {
					downloaded, err := downloadedSelection(selection)
					if err != nil {
						return err
					}
					if err := installDownloadedComponents(cmd, opts.ConfigDir, downloaded, force); err != nil {
						return err
					}
				}
				return nil
			}
			return printBrowse(cmd, providers, source.Selector)
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "print JSON")
	cmd.Flags().BoolVarP(&interactive, "interactive", "i", false, "select and download components interactively")
	cmd.Flags().BoolVarP(&force, "force", "f", false, "overwrite existing local components when interactive")
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

func downloadedSelection(selection browseSelection) ([]downloadedComponent, error) {
	body, err := githubReadFile(selection.Provider.Source, selection.Component.Path)
	if err != nil {
		return nil, err
	}
	id := componentID(body, selection.Component.FileName)
	return []downloadedComponent{{
		ID:       id,
		FileName: id + ".yaml",
		Body:     body,
	}}, nil
}

type browseSelection struct {
	Provider  remoteProvider
	Component remoteComponent
}

type browsePickerItem struct {
	Key         string
	Label       string
	Description string
	Haystack    string
	Selection   browseSelection
}

func runBrowsePicker(input io.Reader, output io.Writer, providers []remoteProvider) ([]browseSelection, error) {
	items := browsePickerItems(providers)
	if len(items) == 0 {
		return nil, nil
	}
	inFile, ok := input.(*os.File)
	if !ok {
		inFile = os.Stdin
	}
	fd := int(inFile.Fd())
	if !term.IsTerminal(fd) {
		return nil, errors.New("--interactive requires terminal stdin")
	}

	oldState, err := term.MakeRaw(fd)
	if err != nil {
		return nil, err
	}
	defer term.Restore(fd, oldState)
	defer fmt.Fprint(output, "\x1b[?25h\x1b[2J\x1b[H")

	selected := map[string]bool{}
	query := ""
	cursor := 0
	reader := bufio.NewReader(inFile)
	fmt.Fprint(output, "\x1b[?25l")

	for {
		filtered := filterBrowseItems(items, query)
		if cursor >= len(filtered) {
			cursor = len(filtered) - 1
		}
		if cursor < 0 {
			cursor = 0
		}

		drawBrowsePicker(output, fd, filtered, selected, query, cursor)
		key, err := readPickerKey(reader, fd)
		if err != nil {
			return nil, err
		}

		switch {
		case key.cancel:
			return nil, errors.New("interactive selection canceled")
		case key.submit:
			return selectedBrowseItems(items, selected), nil
		case key.up:
			if cursor > 0 {
				cursor--
			}
		case key.down:
			if cursor < len(filtered)-1 {
				cursor++
			}
		case key.toggle:
			if len(filtered) > 0 {
				selected[filtered[cursor].Key] = !selected[filtered[cursor].Key]
			}
		case key.backspace:
			if query != "" {
				query = query[:len(query)-1]
				cursor = 0
			}
		case key.char != 0:
			query += string(key.char)
			cursor = 0
		}
	}
}

func browsePickerItems(providers []remoteProvider) []browsePickerItem {
	var items []browsePickerItem
	for _, provider := range providers {
		providerName := firstNonEmpty(provider.Name, provider.ID, provider.SourceString)
		for _, component := range provider.Components {
			label := providerName + "/" + component.ID
			key := provider.SourceString + "#" + component.ID
			haystack := strings.ToLower(label + " " + component.Description + " " + provider.Description)
			items = append(items, browsePickerItem{
				Key:         key,
				Label:       label,
				Description: component.Description,
				Haystack:    haystack,
				Selection: browseSelection{
					Provider:  provider,
					Component: component,
				},
			})
		}
	}
	sort.SliceStable(items, func(i, j int) bool {
		return items[i].Label < items[j].Label
	})
	return items
}

func filterBrowseItems(items []browsePickerItem, query string) []browsePickerItem {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return items
	}
	var out []browsePickerItem
	for _, item := range items {
		if strings.Contains(item.Haystack, query) {
			out = append(out, item)
		}
	}
	return out
}

func drawBrowsePicker(output io.Writer, fd int, items []browsePickerItem, selected map[string]bool, query string, cursor int) {
	width, height, err := term.GetSize(fd)
	if err != nil || width <= 0 {
		width = 100
	}
	if err != nil || height <= 0 {
		height = 24
	}
	maxRows := height - 7
	if maxRows < 5 {
		maxRows = 5
	}

	start := 0
	if cursor >= maxRows {
		start = cursor - maxRows + 1
	}
	end := start + maxRows
	if end > len(items) {
		end = len(items)
	}

	fmt.Fprint(output, "\x1b[2J\x1b[H")
	line(output, "Browse Squire components")
	line(output, "Type to search. Space toggles. Enter downloads. Esc cancels.")
	line(output, "Search: "+query)
	line(output, "")

	if len(items) == 0 {
		line(output, "No matching components.")
		return
	}
	for i := start; i < end; i++ {
		item := items[i]
		prefix := "  "
		if i == cursor {
			prefix = "> "
		}
		box := "[ ]"
		if selected[item.Key] {
			box = "[x]"
		}
		text := prefix + box + " " + item.Label
		if item.Description != "" {
			text += " - " + strings.TrimSuffix(item.Description, ".")
		}
		line(output, truncate(text, width))
	}
	line(output, "")
	line(output, fmt.Sprintf("%d selected", countSelected(selected)))
}

func selectedBrowseItems(items []browsePickerItem, selected map[string]bool) []browseSelection {
	var out []browseSelection
	for _, item := range items {
		if selected[item.Key] {
			out = append(out, item.Selection)
		}
	}
	return out
}
