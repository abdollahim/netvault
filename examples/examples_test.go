package examples

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

// forbidden matches system paths and private or site-specific addresses.
// Committed examples must use documentation addresses and relative paths.
var forbidden = []*regexp.Regexp{
	regexp.MustCompile(`/(var|etc|opt|home)/`),
	regexp.MustCompile(`\b10\.\d{1,3}\.\d{1,3}\.\d{1,3}\b`),
	regexp.MustCompile(`\b192\.168\.\d{1,3}\.\d{1,3}\b`),
	regexp.MustCompile(`\b172\.(1[6-9]|2\d|3[01])\.\d{1,3}\.\d{1,3}\b`),
}

func TestExamplesContainNoSiteData(t *testing.T) {
	files, err := filepath.Glob("*.example.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if len(files) == 0 {
		t.Fatal("no example files found")
	}
	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		for _, re := range forbidden {
			if m := re.Find(data); m != nil {
				t.Errorf("%s contains %q; use placeholders", f, m)
			}
		}
	}
}