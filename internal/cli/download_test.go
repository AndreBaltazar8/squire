package cli

import "testing"

func TestParseComponentSourceShorthand(t *testing.T) {
	source, err := parseComponentSource("AndreBaltazar8/squire-components#svelte")
	if err != nil {
		t.Fatal(err)
	}
	if source.Owner != "AndreBaltazar8" || source.Repo != "squire-components" || source.Selector != "svelte" {
		t.Fatalf("source = %#v", source)
	}
}

func TestParseComponentSourceURLWithRefAndPath(t *testing.T) {
	source, err := parseComponentSource("https://github.com/AndreBaltazar8/squire-components/tree/main/components#go")
	if err != nil {
		t.Fatal(err)
	}
	if source.Owner != "AndreBaltazar8" || source.Repo != "squire-components" || source.Ref != "main" || source.Path != "components" || source.Selector != "go" {
		t.Fatalf("source = %#v", source)
	}
}

func TestComponentIDUsesYAMLID(t *testing.T) {
	id := componentID([]byte("version: 1\nid: svelte\n"), "frontend.yaml")
	if id != "svelte" {
		t.Fatalf("id = %q", id)
	}
}
