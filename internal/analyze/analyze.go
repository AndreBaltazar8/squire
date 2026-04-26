package analyze

import (
	"regexp"
	"sort"
	"strings"

	"squire/internal/guide"
)

var (
	startRe = regexp.MustCompile(`<!--\s*squire:start\s+id=([a-zA-Z0-9_.-]+)(?:\s+required=(true|false))?[^>]*-->`)
	endRe   = regexp.MustCompile(`<!--\s*squire:end\s+id=([a-zA-Z0-9_.-]+)\s*-->`)
)

type Report struct {
	File             string            `json:"file"`
	ExpectedRequired []string          `json:"expected_required"`
	Present          []string          `json:"present"`
	Missing          []string          `json:"missing"`
	UntaggedMatches  []UntaggedHeading `json:"untagged_matches"`
	UnknownTagged    []string          `json:"unknown_tagged"`
	HasManagedMarker bool              `json:"has_managed_marker"`
}

type UntaggedHeading struct {
	SectionID string `json:"section_id"`
	Heading   string `json:"heading"`
}

func File(content, filename string, expected []guide.Section) Report {
	expectedIDs := map[string]bool{}
	expectedTitles := map[string]guide.Section{}
	required := []string{}
	for _, section := range expected {
		expectedIDs[section.ID] = true
		expectedTitles[slug(section.Title)] = section
		if section.Required {
			required = append(required, section.ID)
		}
	}

	presentSet := map[string]bool{}
	for _, match := range startRe.FindAllStringSubmatch(content, -1) {
		presentSet[match[1]] = true
	}

	endIDs := map[string]bool{}
	for _, match := range endRe.FindAllStringSubmatch(content, -1) {
		endIDs[match[1]] = true
	}
	for id := range presentSet {
		if !endIDs[id] {
			delete(presentSet, id)
		}
	}

	present := sortedKeys(presentSet)
	missing := []string{}
	for _, id := range required {
		if !presentSet[id] {
			missing = append(missing, id)
		}
	}

	unknown := []string{}
	for _, id := range present {
		if !expectedIDs[id] {
			unknown = append(unknown, id)
		}
	}

	untagged := untaggedMatches(content, expectedTitles, presentSet)

	return Report{
		File:             filename,
		ExpectedRequired: required,
		Present:          present,
		Missing:          missing,
		UntaggedMatches:  untagged,
		UnknownTagged:    unknown,
		HasManagedMarker: strings.Contains(content, "squire:managed"),
	}
}

func ExpectedForTarget(target string) []guide.Section {
	return guide.ExpectedForTarget(target)
}

func IsComplete(report Report) bool {
	return len(report.Missing) == 0
}

func untaggedMatches(content string, expectedTitles map[string]guide.Section, present map[string]bool) []UntaggedHeading {
	var out []UntaggedHeading
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "#") {
			continue
		}
		title := strings.TrimSpace(strings.TrimLeft(line, "#"))
		if title == "" {
			continue
		}
		if section, ok := expectedTitles[slug(title)]; ok && !present[section.ID] {
			out = append(out, UntaggedHeading{
				SectionID: section.ID,
				Heading:   title,
			})
		}
	}
	return out
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
