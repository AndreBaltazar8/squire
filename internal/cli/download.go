package cli

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"squire/internal/config"
)

type componentSource struct {
	Owner    string
	Repo     string
	Ref      string
	Path     string
	Selector string
}

type githubContent struct {
	Name        string          `json:"name"`
	Path        string          `json:"path"`
	Type        string          `json:"type"`
	DownloadURL string          `json:"download_url"`
	Content     string          `json:"content"`
	Encoding    string          `json:"encoding"`
	Items       []githubContent `json:"-"`
}

func newDownloadCommand(opts *rootOptions) *cobra.Command {
	var force bool
	cmd := &cobra.Command{
		Use:   "download <github-source[#component]>",
		Short: "Download components into the Squire config directory",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			source, err := parseComponentSource(args[0])
			if err != nil {
				return err
			}
			configDir := opts.ConfigDir
			if configDir == "" {
				configDir, err = config.DefaultConfigDir()
				if err != nil {
					return err
				}
			}
			if err := config.EnsureDefaults(configDir); err != nil {
				return err
			}

			downloaded, err := downloadComponents(source)
			if err != nil {
				return err
			}
			if len(downloaded) == 0 {
				if source.Selector != "" {
					return fmt.Errorf("component %q not found in %s/%s", source.Selector, source.Owner, source.Repo)
				}
				return fmt.Errorf("no components found in %s/%s", source.Owner, source.Repo)
			}

			componentsDir := filepath.Join(configDir, config.ComponentsDir)
			if err := os.MkdirAll(componentsDir, 0o755); err != nil {
				return err
			}
			for _, component := range downloaded {
				target := filepath.Join(componentsDir, component.FileName)
				if !force {
					if existing, err := os.ReadFile(target); err == nil {
						if string(existing) == string(component.Body) {
							fmt.Fprintf(cmd.OutOrStdout(), "kept %s\n", target)
							continue
						}
						return fmt.Errorf("%s already exists and differs; rerun with --force to overwrite", target)
					} else if !errors.Is(err, os.ErrNotExist) {
						return err
					}
				}
				if err := os.WriteFile(target, component.Body, 0o644); err != nil {
					return err
				}
				fmt.Fprintf(cmd.OutOrStdout(), "downloaded %s\n", target)
			}
			return nil
		},
	}
	cmd.Flags().BoolVarP(&force, "force", "f", false, "overwrite existing local components")
	return cmd
}

type downloadedComponent struct {
	ID       string
	FileName string
	Body     []byte
}

func parseComponentSource(raw string) (componentSource, error) {
	source, selector, _ := strings.Cut(strings.TrimSpace(raw), "#")
	if source == "" {
		return componentSource{}, errors.New("component source is required")
	}

	out := componentSource{Selector: strings.TrimSpace(selector)}
	if parsed, err := url.Parse(source); err == nil && parsed.Scheme != "" {
		if parsed.Host != "github.com" {
			return componentSource{}, errors.New("only github.com component URLs are supported")
		}
		parts := splitPath(parsed.Path)
		if len(parts) < 2 {
			return componentSource{}, fmt.Errorf("invalid GitHub component URL %q", raw)
		}
		out.Owner = parts[0]
		out.Repo = strings.TrimSuffix(parts[1], ".git")
		if len(parts) >= 5 && parts[2] == "tree" {
			out.Ref = parts[3]
			out.Path = strings.Join(parts[4:], "/")
		}
		return out, nil
	}

	parts := splitPath(source)
	if len(parts) < 2 {
		return componentSource{}, fmt.Errorf("expected owner/repo or github.com URL, got %q", raw)
	}
	out.Owner = parts[0]
	out.Repo = strings.TrimSuffix(parts[1], ".git")
	if len(parts) > 2 {
		out.Path = strings.Join(parts[2:], "/")
	}
	return out, nil
}

func splitPath(value string) []string {
	value = strings.Trim(value, "/")
	if value == "" {
		return nil
	}
	parts := strings.Split(value, "/")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func downloadComponents(source componentSource) ([]downloadedComponent, error) {
	var candidates []githubContent
	searchPaths := []string{source.Path}
	if source.Path == "" {
		searchPaths = []string{"components", ""}
	}
	for _, dir := range searchPaths {
		items, err := githubList(source, dir)
		if err != nil {
			if isGithubNotFound(err) {
				continue
			}
			return nil, err
		}
		for _, item := range items {
			if item.Type == "file" && isYAML(item.Name) {
				candidates = append(candidates, item)
			}
		}
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		return candidates[i].Path < candidates[j].Path
	})

	var out []downloadedComponent
	for _, candidate := range candidates {
		body, err := githubReadFile(source, candidate.Path)
		if err != nil {
			return nil, err
		}
		id := componentID(body, candidate.Name)
		if source.Selector != "" && source.Selector != id && source.Selector != strings.TrimSuffix(candidate.Name, filepath.Ext(candidate.Name)) {
			continue
		}
		out = append(out, downloadedComponent{
			ID:       id,
			FileName: id + ".yaml",
			Body:     body,
		})
	}
	return out, nil
}

func githubList(source componentSource, dir string) ([]githubContent, error) {
	endpoint := githubAPIPath(source, dir)
	var raw json.RawMessage
	if err := githubGET(endpoint, &raw); err != nil {
		return nil, err
	}
	var items []githubContent
	if err := json.Unmarshal(raw, &items); err == nil {
		return items, nil
	}
	var single githubContent
	if err := json.Unmarshal(raw, &single); err != nil {
		return nil, err
	}
	return []githubContent{single}, nil
}

func githubReadFile(source componentSource, filePath string) ([]byte, error) {
	var content githubContent
	if err := githubGET(githubAPIPath(source, filePath), &content); err != nil {
		return nil, err
	}
	if content.Encoding == "base64" && content.Content != "" {
		compact := strings.ReplaceAll(content.Content, "\n", "")
		return base64.StdEncoding.DecodeString(compact)
	}
	if content.DownloadURL != "" {
		resp, err := http.Get(content.DownloadURL)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return nil, fmt.Errorf("GET %s: %s", content.DownloadURL, resp.Status)
		}
		return io.ReadAll(resp.Body)
	}
	return nil, fmt.Errorf("GitHub content %s has no downloadable body", filePath)
}

func githubAPIPath(source componentSource, filePath string) string {
	escapedPath := strings.Trim(path.Clean("/"+filePath), "/")
	endpoint := fmt.Sprintf("https://api.github.com/repos/%s/%s/contents", source.Owner, source.Repo)
	if escapedPath != "" {
		endpoint += "/" + escapedPath
	}
	if source.Ref != "" {
		endpoint += "?ref=" + url.QueryEscape(source.Ref)
	}
	return endpoint
}

func githubGET(endpoint string, target any) error {
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "squire")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &githubStatusError{StatusCode: resp.StatusCode, Status: resp.Status}
	}
	return json.NewDecoder(resp.Body).Decode(target)
}

type githubStatusError struct {
	StatusCode int
	Status     string
}

func (err *githubStatusError) Error() string {
	return "GitHub API returned " + err.Status
}

func isGithubNotFound(err error) bool {
	var statusErr *githubStatusError
	return errors.As(err, &statusErr) && statusErr.StatusCode == http.StatusNotFound
}

func isYAML(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	return ext == ".yaml" || ext == ".yml"
}

func componentID(body []byte, filename string) string {
	var component config.Component
	if err := yaml.Unmarshal(body, &component); err == nil && strings.TrimSpace(component.ID) != "" {
		return strings.TrimSpace(component.ID)
	}
	return strings.TrimSuffix(filename, filepath.Ext(filename))
}
