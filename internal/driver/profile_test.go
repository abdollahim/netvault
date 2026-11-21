package driver

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const committedProfile = "../../profiles/cisco-ios.yaml"

const validProfile = `name: test
transport:
  key_exchanges: [diffie-hellman-group14-sha1]
  ciphers: [aes128-cbc]
  macs: [hmac-sha1]
  host_key_algorithms: [ssh-rsa]
prompt:
  user_exec: '^\S+>\s*$'
  privileged: '^\S+#\s*$'
  password: '(?i)password:\s*$'
  pager: '--More--'
  pager_answer: ' '
enable:
  command: enable
timeouts:
  connect_seconds: 20
  prompt_seconds: 30
  command_seconds: 600
`

func writeProfile(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "profile.yaml")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestCommittedProfileLoads(t *testing.T) {
	p, err := LoadProfile(committedProfile)
	if err != nil {
		t.Fatal(err)
	}
	if p.Name != "cisco-ios" {
		t.Fatalf("name = %q, want cisco-ios", p.Name)
	}
	if p.PromptTimeout() <= 0 || p.CommandTimeout() <= p.PromptTimeout() {
		t.Fatal("timeouts not loaded as expected")
	}
}

func TestValidProfileLoads(t *testing.T) {
	if _, err := LoadProfile(writeProfile(t, validProfile)); err != nil {
		t.Fatal(err)
	}
}

func TestUnknownFieldIsRejected(t *testing.T) {
	_, err := LoadProfile(writeProfile(t, validProfile+"\nprompts: x\n"))
	if err == nil || !strings.Contains(err.Error(), "prompts") {
		t.Fatalf("misspelled field not reported: %v", err)
	}
}

func TestBrokenPatternNamesField(t *testing.T) {
	bad := strings.Replace(validProfile, `'^\S+>\s*$'`, `'[unclosed'`, 1)
	_, err := LoadProfile(writeProfile(t, bad))
	if err == nil || !strings.Contains(err.Error(), "prompt.user_exec") {
		t.Fatalf("bad pattern not reported with its field: %v", err)
	}
}

func TestMissingAlgorithmsRejected(t *testing.T) {
	bad := strings.Replace(validProfile,
		"key_exchanges: [diffie-hellman-group14-sha1]", "key_exchanges: []", 1)
	_, err := LoadProfile(writeProfile(t, bad))
	if err == nil || !strings.Contains(err.Error(), "key_exchanges") {
		t.Fatalf("missing key exchange not reported: %v", err)
	}
}