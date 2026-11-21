package transport

import (
	"fmt"
	"net"
	"time"

	"golang.org/x/crypto/ssh"

	"netvault/internal/profile"
)

// Algorithm sets for older network devices. The Go SSH library leaves these
// out of its defaults, so they are enabled explicitly. They are used by the
// in-process test device and by LegacyClientConfig.
var (
	legacyKeyExchanges = []string{
		"diffie-hellman-group14-sha1",
		"diffie-hellman-group1-sha1",
	}
	legacyCiphers = []string{
		"aes128-cbc", "aes192-cbc", "aes256-cbc", "3des-cbc",
	}
	legacyMACs = []string{
		"hmac-sha1",
	}
	legacyHostKeyAlgorithms = []string{"ssh-rsa"}
)

// LegacyClientConfig returns a client configuration that offers only legacy
// algorithms. Used by tests; the spike reads its settings from the profile.
func LegacyClientConfig(user, password string, hostKey ssh.HostKeyCallback) *ssh.ClientConfig {
	return &ssh.ClientConfig{
		User:              user,
		Auth:              []ssh.AuthMethod{ssh.Password(password)},
		HostKeyCallback:   hostKey,
		HostKeyAlgorithms: legacyHostKeyAlgorithms,
		Timeout:           20 * time.Second,
		Config: ssh.Config{
			KeyExchanges: legacyKeyExchanges,
			Ciphers:      legacyCiphers,
			MACs:         legacyMACs,
		},
	}
}

// ClientConfigFromProfile builds a client configuration whose algorithm lists
// come from the brand profile, so new brands need only a profile file.
func ClientConfigFromProfile(p *profile.Profile, user, password string, hostKey ssh.HostKeyCallback) *ssh.ClientConfig {
	return &ssh.ClientConfig{
		User:              user,
		Auth:              []ssh.AuthMethod{ssh.Password(password)},
		HostKeyCallback:   hostKey,
		HostKeyAlgorithms: p.Transport.HostKeyAlgorithms,
		Timeout:           p.ConnectTimeout(),
		Config: ssh.Config{
			KeyExchanges: p.Transport.KeyExchanges,
			Ciphers:      p.Transport.Ciphers,
			MACs:         p.Transport.MACs,
		},
	}
}

// Dial opens an SSH connection to addr (host:port) using cfg.
func Dial(addr string, cfg *ssh.ClientConfig) (*ssh.Client, error) {
	conn, err := net.DialTimeout("tcp", addr, cfg.Timeout)
	if err != nil {
		return nil, fmt.Errorf("dial %s: %w", addr, err)
	}
	c, chans, reqs, err := ssh.NewClientConn(conn, addr, cfg)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("ssh to %s: %w", addr, err)
	}
	return ssh.NewClient(c, chans, reqs), nil
}