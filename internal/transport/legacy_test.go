package transport

import (
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"io"
	"net"
	"path/filepath"
	"testing"
	"time"

	"golang.org/x/crypto/ssh"
)

const fakeShowVersion = "Cisco IOS Software, test image\nhostname test-switch\n"

// startLegacyDevice runs an in-process SSH server that offers only the legacy
// algorithms and answers exec requests with fakeShowVersion.
func startLegacyDevice(t *testing.T) string {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	signer, err := ssh.NewSignerFromKey(key)
	if err != nil {
		t.Fatal(err)
	}
	cfg := &ssh.ServerConfig{
		PasswordCallback: func(c ssh.ConnMetadata, pw []byte) (*ssh.Permissions, error) {
			if c.User() == "ro" && string(pw) == "secret" {
				return nil, nil
			}
			return nil, fmt.Errorf("access denied")
		},
		Config: ssh.Config{
			KeyExchanges: legacyKeyExchanges,
			Ciphers:      legacyCiphers,
			MACs:         legacyMACs,
		},
	}
	cfg.AddHostKey(signer)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ln.Close() })
	go func() {
		for {
			nc, err := ln.Accept()
			if err != nil {
				return
			}
			go serveExec(nc, cfg)
		}
	}()
	return ln.Addr().String()
}

func serveExec(nc net.Conn, cfg *ssh.ServerConfig) {
	conn, chans, reqs, err := ssh.NewServerConn(nc, cfg)
	if err != nil {
		nc.Close()
		return
	}
	defer conn.Close()
	go ssh.DiscardRequests(reqs)
	for nch := range chans {
		if nch.ChannelType() != "session" {
			nch.Reject(ssh.UnknownChannelType, "unsupported")
			continue
		}
		ch, creqs, err := nch.Accept()
		if err != nil {
			return
		}
		go func() {
			defer ch.Close()
			for req := range creqs {
				if req.Type != "exec" {
					req.Reply(false, nil)
					continue
				}
				req.Reply(true, nil)
				io.WriteString(ch, fakeShowVersion)
				ch.SendRequest("exit-status", false, []byte{0, 0, 0, 0})
				return
			}
		}()
	}
}

func TestLegacyClientRunsCommand(t *testing.T) {
	addr := startLegacyDevice(t)
	pin := filepath.Join(t.TempDir(), "device.pin")
	client, err := Dial(addr, LegacyClientConfig("ro", "secret", PinnedHostKey(pin)))
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer client.Close()
	sess, err := client.NewSession()
	if err != nil {
		t.Fatal(err)
	}
	defer sess.Close()
	out, err := sess.Output("show version")
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != fakeShowVersion {
		t.Fatalf("got %q, want %q", out, fakeShowVersion)
	}
}

func TestModernClientIsRefusedByLegacyDevice(t *testing.T) {
	addr := startLegacyDevice(t)
	cfg := &ssh.ClientConfig{
		User:            "ro",
		Auth:            []ssh.AuthMethod{ssh.Password("secret")},
		HostKeyCallback: PinnedHostKey(filepath.Join(t.TempDir(), "device.pin")),
		Timeout:         10 * time.Second,
	}
	client, err := Dial(addr, cfg)
	if err == nil {
		client.Close()
		t.Fatal("modern-only client connected to a legacy-only device")
	}
}

func TestWrongPasswordIsRefused(t *testing.T) {
	addr := startLegacyDevice(t)
	pin := filepath.Join(t.TempDir(), "device.pin")
	client, err := Dial(addr, LegacyClientConfig("ro", "wrong", PinnedHostKey(pin)))
	if err == nil {
		client.Close()
		t.Fatal("wrong password accepted")
	}
}