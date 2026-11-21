package transport

import (
	"strings"
	"testing"
)

func TestPromptDetection(t *testing.T) {
	for _, p := range []string{"sw1#", "sw1>", "DC01-RC10_WS-C6509-E_Core#", "sw1(config)#"} {
		if !promptLine.MatchString(p) {
			t.Errorf("prompt %q not detected", p)
		}
	}
	for _, l := range []string{"interface GigabitEthernet1/1", "Password: ", "--More-- ", "end"} {
		if promptLine.MatchString(l) {
			t.Errorf("%q wrongly detected as a prompt", l)
		}
	}
}

func TestPagerMarkerRemoved(t *testing.T) {
	in := "line one\n --More-- \x08\x08\x08\x08\x08\x08\x08\x08\x08         \x08\x08\nline two"
	got := pagerStrip.ReplaceAllString(in, "")
	if strings.Contains(got, "--More--") || strings.Contains(got, "\x08") {
		t.Fatalf("pager marker left in output: %q", got)
	}
	if !strings.Contains(got, "line one") || !strings.Contains(got, "line two") {
		t.Fatalf("content lost: %q", got)
	}
}

func TestLooksComplete(t *testing.T) {
	complete := "Building configuration...\n\nCurrent configuration : 120 bytes\nhostname sw1\n!\nend\n"
	if !LooksComplete(complete) {
		t.Error("complete config rejected")
	}
	truncated := "Building configuration...\n\nCurrent configuration : 120 bytes\nhostname sw1\n"
	if LooksComplete(truncated) {
		t.Error("truncated config accepted")
	}
}