package analyze

import (
	"testing"

	"squire/internal/guide"
)

func TestFileReportsTaggedAndMissingSections(t *testing.T) {
	sections := []guide.Section{
		{ID: "technology-stack", Title: "Technology Stack", Required: true},
		{ID: "commands", Title: "Commands", Required: true},
	}
	body := `<!-- squire:managed file=AGENTS.md -->
<!-- squire:start id=technology-stack required=true -->
## Technology Stack

- Go
<!-- squire:end id=technology-stack -->
## Commands
`

	report := File(body, "AGENTS.md", sections)

	if len(report.Present) != 1 || report.Present[0] != "technology-stack" {
		t.Fatalf("present = %#v", report.Present)
	}
	if len(report.Missing) != 1 || report.Missing[0] != "commands" {
		t.Fatalf("missing = %#v", report.Missing)
	}
	if len(report.UntaggedMatches) != 1 || report.UntaggedMatches[0].SectionID != "commands" {
		t.Fatalf("untagged = %#v", report.UntaggedMatches)
	}
	if !report.HasManagedMarker {
		t.Fatal("expected managed marker")
	}
}
