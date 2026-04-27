package cli

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/AndreBaltazar8/squire/internal/config"
)

const defaultProviderSource = "AndreBaltazar8/squire-components"

type componentSource struct {
	Owner    string
	Repo     string
	Ref      string
	Path     string
	Selector string
}

func (source componentSource) String() string {
	value := source.Owner + "/" + source.Repo
	if source.Path != "" {
		value += "/" + strings.Trim(source.Path, "/")
	}
	if source.Ref != "" {
		value += "@" + source.Ref
	}
	return value
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

type providerManifest struct {
	Version     int    `yaml:"version" json:"version"`
	ID          string `yaml:"id" json:"id"`
	Name        string `yaml:"name" json:"name"`
	Description string `yaml:"description" json:"description"`
	Components  string `yaml:"components" json:"components"`
}

type providerIndex struct {
	Version   int                 `yaml:"version" json:"version"`
	Providers []providerReference `yaml:"providers" json:"providers"`
}

type componentCatalog struct {
	Version     int               `yaml:"version" json:"version"`
	GeneratedAt string            `yaml:"generated_at" json:"generated_at"`
	Providers   []catalogProvider `yaml:"providers" json:"providers"`
}

type catalogProvider struct {
	ID            string             `yaml:"id" json:"id"`
	Name          string             `yaml:"name" json:"name"`
	Description   string             `yaml:"description" json:"description"`
	Source        string             `yaml:"source" json:"source"`
	ComponentsDir string             `yaml:"components_dir" json:"components_dir"`
	Components    []catalogComponent `yaml:"components" json:"components"`
}

type catalogComponent struct {
	ID          string `yaml:"id" json:"id"`
	Description string `yaml:"description" json:"description"`
	Path        string `yaml:"path" json:"path"`
}

type providerReference struct {
	ID          string `yaml:"id" json:"id"`
	Name        string `yaml:"name" json:"name"`
	Source      string `yaml:"source" json:"source"`
	Description string `yaml:"description" json:"description"`
}

type remoteProvider struct {
	ID            string
	Name          string
	Description   string
	Source        componentSource
	SourceString  string
	ComponentsDir string
	Components    []remoteComponent
}

type remoteComponent struct {
	ID          string
	FileName    string
	Path        string
	Description string
	Body        []byte
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

			return installDownloadedComponents(cmd, configDir, downloaded, force)
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
	provider, err := loadProvider(source, providerReference{})
	if err != nil {
		return nil, err
	}
	var out []downloadedComponent
	for _, component := range provider.Components {
		if source.Selector != "" && source.Selector != component.ID && source.Selector != strings.TrimSuffix(component.FileName, filepath.Ext(component.FileName)) {
			continue
		}
		out = append(out, downloadedComponent{
			ID:       component.ID,
			FileName: component.ID + ".yaml",
			Body:     component.Body,
		})
	}
	return out, nil
}

func browseProviders(source componentSource) ([]remoteProvider, error) {
	if providers, ok, err := readComponentCatalog(source); err != nil {
		return nil, err
	} else if ok {
		return filterProviderComponents(providers, source.Selector), nil
	}

	refs, ok, err := readProviderIndex(source)
	if err != nil {
		return nil, err
	}
	if !ok {
		provider, err := loadProvider(source, providerReference{})
		if err != nil {
			return nil, err
		}
		return filterProviderComponents([]remoteProvider{provider}, source.Selector), nil
	}

	var providers []remoteProvider
	seen := map[string]bool{}
	for _, ref := range refs {
		providerSource := source
		if strings.TrimSpace(ref.Source) != "" {
			providerSource, err = parseComponentSource(ref.Source)
			if err != nil {
				return nil, err
			}
		}
		providerSource.Selector = source.Selector
		key := providerSource.String()
		if seen[key] {
			continue
		}
		seen[key] = true
		provider, err := loadProvider(providerSource, ref)
		if err != nil {
			return nil, err
		}
		providers = append(providers, provider)
	}
	return filterProviderComponents(providers, source.Selector), nil
}

func readComponentCatalog(source componentSource) ([]remoteProvider, bool, error) {
	if strings.TrimSpace(source.Path) != "" {
		return nil, false, nil
	}
	body, err := githubReadFile(source, "index.yaml")
	if err != nil {
		if isGithubNotFound(err) {
			return nil, false, nil
		}
		return nil, false, err
	}
	var catalog componentCatalog
	if err := yaml.Unmarshal(body, &catalog); err != nil {
		return nil, false, err
	}
	providers, err := catalogProviders(source, catalog)
	if err != nil {
		return nil, false, err
	}
	return providers, true, nil
}

func catalogProviders(source componentSource, catalog componentCatalog) ([]remoteProvider, error) {
	providers := make([]remoteProvider, 0, len(catalog.Providers))
	for _, item := range catalog.Providers {
		providerSource := source
		if item.Source != "" {
			parsed, err := parseComponentSource(item.Source)
			if err != nil {
				return nil, err
			}
			providerSource = parsed
		}
		providerSource.Selector = source.Selector
		provider := remoteProvider{
			ID:            item.ID,
			Name:          item.Name,
			Description:   item.Description,
			Source:        providerSource,
			SourceString:  providerSource.String(),
			ComponentsDir: firstNonEmpty(item.ComponentsDir, "components"),
		}
		for _, component := range item.Components {
			provider.Components = append(provider.Components, remoteComponent{
				ID:          component.ID,
				FileName:    filepath.Base(component.Path),
				Path:        component.Path,
				Description: component.Description,
			})
		}
		sort.SliceStable(provider.Components, func(i, j int) bool {
			return provider.Components[i].ID < provider.Components[j].ID
		})
		providers = append(providers, provider)
	}
	sort.SliceStable(providers, func(i, j int) bool {
		return providers[i].ID < providers[j].ID
	})
	return providers, nil
}

func loadProvider(source componentSource, ref providerReference) (remoteProvider, error) {
	manifest, ok, err := readProviderManifest(source)
	if err != nil {
		return remoteProvider{}, err
	}
	if !ok {
		manifest = providerManifest{
			ID:          source.Repo,
			Name:        source.Repo,
			Description: ref.Description,
			Components:  "components",
		}
	}
	if manifest.ID == "" {
		manifest.ID = firstNonEmpty(ref.ID, source.Repo)
	}
	if manifest.Name == "" {
		manifest.Name = firstNonEmpty(ref.Name, manifest.ID)
	}
	if manifest.Description == "" {
		manifest.Description = ref.Description
	}
	componentsDir := componentDir(source, manifest)
	components, err := readRemoteComponents(source, componentsDir)
	if err != nil {
		if isGithubNotFound(err) {
			return remoteProvider{
				ID:            manifest.ID,
				Name:          manifest.Name,
				Description:   manifest.Description,
				Source:        source,
				SourceString:  source.String(),
				ComponentsDir: componentsDir,
			}, nil
		}
		return remoteProvider{}, err
	}
	return remoteProvider{
		ID:            manifest.ID,
		Name:          manifest.Name,
		Description:   manifest.Description,
		Source:        source,
		SourceString:  source.String(),
		ComponentsDir: componentsDir,
		Components:    components,
	}, nil
}

func readProviderManifest(source componentSource) (providerManifest, bool, error) {
	body, err := githubReadFile(source, "provider.yaml")
	if err != nil {
		if isGithubNotFound(err) {
			return providerManifest{}, false, nil
		}
		return providerManifest{}, false, err
	}
	var manifest providerManifest
	if err := yaml.Unmarshal(body, &manifest); err != nil {
		return providerManifest{}, false, err
	}
	return manifest, true, nil
}

func readProviderIndex(source componentSource) ([]providerReference, bool, error) {
	body, err := githubReadFile(source, "providers.yaml")
	if err != nil {
		if isGithubNotFound(err) {
			return nil, false, nil
		}
		return nil, false, err
	}
	var index providerIndex
	if err := yaml.Unmarshal(body, &index); err != nil {
		return nil, false, err
	}
	return index.Providers, true, nil
}

func readRemoteComponents(source componentSource, dir string) ([]remoteComponent, error) {
	items, err := githubList(source, dir)
	if err != nil {
		return nil, err
	}
	var candidates []githubContent
	for _, item := range items {
		if item.Type == "file" && isYAML(item.Name) {
			candidates = append(candidates, item)
		}
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		return candidates[i].Path < candidates[j].Path
	})

	var components []remoteComponent
	for _, candidate := range candidates {
		body, err := githubReadFile(source, candidate.Path)
		if err != nil {
			return nil, err
		}
		component := componentMeta(body, candidate.Name)
		components = append(components, remoteComponent{
			ID:          component.ID,
			FileName:    candidate.Name,
			Path:        candidate.Path,
			Description: component.Description,
			Body:        body,
		})
	}
	sort.SliceStable(components, func(i, j int) bool {
		return components[i].ID < components[j].ID
	})
	return components, nil
}

func componentDir(source componentSource, manifest providerManifest) string {
	if strings.TrimSpace(source.Path) != "" {
		return strings.Trim(strings.TrimSpace(source.Path), "/")
	}
	if strings.TrimSpace(manifest.Components) != "" {
		return strings.Trim(strings.TrimSpace(manifest.Components), "/")
	}
	return "components"
}

func filterProviderComponents(providers []remoteProvider, selector string) []remoteProvider {
	selector = strings.TrimSpace(selector)
	if selector == "" {
		return providers
	}
	var out []remoteProvider
	for _, provider := range providers {
		filtered := provider.Components[:0]
		for _, component := range provider.Components {
			if selector == component.ID || selector == strings.TrimSuffix(component.FileName, filepath.Ext(component.FileName)) {
				filtered = append(filtered, component)
			}
		}
		provider.Components = filtered
		if len(provider.Components) > 0 {
			out = append(out, provider)
		}
	}
	return out
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
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

func installDownloadedComponents(cmd *cobra.Command, configDir string, downloaded []downloadedComponent, force bool) error {
	if configDir == "" {
		var err error
		configDir, err = config.DefaultConfigDir()
		if err != nil {
			return err
		}
	}
	if err := config.EnsureDefaults(configDir); err != nil {
		return err
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
	if token := githubToken(); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return &githubStatusError{StatusCode: resp.StatusCode, Status: resp.Status, Body: strings.TrimSpace(string(body))}
	}
	return json.NewDecoder(resp.Body).Decode(target)
}

type githubStatusError struct {
	StatusCode int
	Status     string
	Body       string
}

func (err *githubStatusError) Error() string {
	if err.Body != "" {
		return "GitHub API returned " + err.Status + ": " + err.Body
	}
	return "GitHub API returned " + err.Status
}

var (
	cachedGitHubToken  string
	checkedGitHubToken bool
)

func githubToken() string {
	if token := strings.TrimSpace(os.Getenv("GH_TOKEN")); token != "" {
		return token
	}
	if token := strings.TrimSpace(os.Getenv("GITHUB_TOKEN")); token != "" {
		return token
	}
	if checkedGitHubToken {
		return cachedGitHubToken
	}
	checkedGitHubToken = true
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, "gh", "auth", "token").Output()
	if err != nil {
		return ""
	}
	cachedGitHubToken = strings.TrimSpace(string(out))
	return cachedGitHubToken
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
	return componentMeta(body, filename).ID
}

func componentMeta(body []byte, filename string) config.Component {
	var component config.Component
	if err := yaml.Unmarshal(body, &component); err != nil {
		component = config.Component{}
	}
	if strings.TrimSpace(component.ID) == "" {
		component.ID = strings.TrimSuffix(filename, filepath.Ext(filename))
	}
	component.ID = strings.TrimSpace(component.ID)
	return component
}
