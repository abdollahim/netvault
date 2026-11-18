package transport

import (
	"fmt"
	"net"
	"os"
	"strings"

	"golang.org/x/crypto/ssh"
)

// PinnedHostKey trusts a device's host key by fingerprint. The first key seen
// is written to pinFile. A later key that differs is refused, so a changed
// key stops the connection instead of being accepted silently.
func PinnedHostKey(pinFile string) ssh.HostKeyCallback {
	return func(hostname string, _ net.Addr, key ssh.PublicKey) error {
		got := ssh.FingerprintSHA256(key)
		stored, err := os.ReadFile(pinFile)
		if os.IsNotExist(err) {
			return os.WriteFile(pinFile, []byte(got+"\n"), 0o600)
		}
		if err != nil {
			return fmt.Errorf("read host key pin: %w", err)
		}
		want := strings.TrimSpace(string(stored))
		if want != got {
			return fmt.Errorf("host key for %s changed: pinned %s, got %s", hostname, want, got)
		}
		return nil
	}
}