package transport

import (
	"regexp"
	"testing"
)

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

func TestProfilePromptPatterns(t *testing.T) {
	p := testProfile(t)
	userRe := regexp.MustCompile(p.Prompt.UserExec)
	privRe := regexp.MustCompile(p.Prompt.Privileged)

	for _, s := range []string{"sw-core-01>", "sw1>"} {
		if !userRe.MatchString(s) {
			t.Errorf("user prompt %q not matched", s)
		}
	}
	for _, s := range []string{"sw-core-01#", "sw1(config)#"} {
		if !privRe.MatchString(s) {
			t.Errorf("privileged prompt %q not matched", s)
		}
	}
	for _, s := range []string{"interface GigabitEthernet1/1", "--More-- "} {
		if userRe.MatchString(s) || privRe.MatchString(s) {
			t.Errorf("%q wrongly matched as a prompt", s)
		}
	}
}