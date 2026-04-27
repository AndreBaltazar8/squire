package analyze

import (
	"sort"
	"strings"

	"github.com/AndreBaltazar8/squire/internal/guide"
)

type Report struct {
	File               string            `json:"file"`
	ExpectedComponents []string          `json:"expected_components,omitempty"`
	ExpectedRequired   []string          `json:"expected_required"`
	Present            []string          `json:"present"`
	Missing            []string          `json:"missing"`
	OutOfDate          bool              `json:"out_of_date"`
	UntaggedMatches    []UntaggedHeading `json:"untagged_matches"`
}

type UntaggedHeading struct {
	SectionID string `json:"section_id"`
	Heading   string `json:"heading"`
}

func File(content, filename string, expected []guide.Section) Report {
	expectedTitles := map[string]guide.Section{}
	required := []string{}
	for _, section := range expected {
		expectedTitles[slug(section.Title)] = section
		if section.Required {
			required = append(required, section.ID)
		}
	}

	presentSet := headingMatches(content, expectedTitles)

	present := sortedKeys(presentSet)
	missing := []string{}
	for _, id := range required {
		if !presentSet[id] {
			missing = append(missing, id)
		}
	}

	return Report{
		File:             filename,
		ExpectedRequired: required,
		Present:          present,
		Missing:          missing,
		UntaggedMatches:  nil,
	}
}

func ExpectedForTarget(target string) []guide.Section {
	return guide.ExpectedForTarget(target)
}

func IsComplete(report Report) bool {
	return len(report.Missing) == 0
}

func headingMatches(content string, expectedTitles map[string]guide.Section) map[string]bool {
	out := map[string]bool{}
	for _, line := range strings.Split(content, "\n") {
		_, title, ok := markdownHeading(line)
		if !ok {
			continue
		}
		if section, ok := expectedTitles[slug(title)]; ok {
			out[section.ID] = true
		}
	}
	return out
}

func markdownHeading(line string) (int, string, bool) {
	line = strings.TrimSpace(line)
	level := 0
	for level < len(line) && level < 6 && line[level] == '#' {
		level++
	}
	if level == 0 || level >= len(line) || line[level] != ' ' {
		return 0, "", false
	}
	title := strings.TrimSpace(line[level+1:])
	title = strings.TrimSpace(strings.TrimRight(title, "#"))
	if title == "" {
		return 0, "", false
	}
	return level, title, true
}

func sortedKeys(values map[string]bool) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func slug(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var out strings.Builder
	lastDash := false
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			out.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			out.WriteRune('-')
			lastDash = true
		}
	}
	return strings.Trim(out.String(), "-")
}
