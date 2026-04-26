package analyze

import (
	"testing"

	"squire/internal/guide"
)

func TestFileReportsMissingSections(t *testing.T) {
	sections := []guide.Section{
		{ID: "technology-stack", Title: "Technology Stack", Required: true},
		{ID: "commands", Title: "Commands", Required: true},
	}
	body := `# Demo Agent Guide

## Technology Stack

- Go

## Other
`

	report := File(body, "AGENTS.md", sections)

	if len(report.Present) != 1 || report.Present[0] != "technology-stack" {
		t.Fatalf("present = %#v", report.Present)
	}
	if len(report.Missing) != 1 || report.Missing[0] != "commands" {
		t.Fatalf("missing = %#v", report.Missing)
	}
}

func TestFileReportsMarkerlessHeadingSections(t *testing.T) {
	sections := []guide.Section{
		{ID: "technology-stack", Title: "Technology Stack", Required: true},
		{ID: "commands", Title: "Commands", Required: true},
	}
	body := `# Demo Agent Guide

## Technology Stack

- Go

## Commands

- go test ./...
`

	report := File(body, "AGENTS.md", sections)

	if len(report.Present) != 2 || report.Present[0] != "commands" || report.Present[1] != "technology-stack" {
		t.Fatalf("present = %#v", report.Present)
	}
	if len(report.Missing) != 0 {
		t.Fatalf("missing = %#v", report.Missing)
	}
}
