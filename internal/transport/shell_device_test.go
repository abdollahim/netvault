package transport

import (
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"fmt"
	"io"
	"net"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"netvault/internal/profile"
	"golang.org/x/crypto/ssh"
)

// testProfile returns the committed brand profile, so the fake-device tests
// use the same settings a real switch does.
func testProfile(t *testing.T) *profile.Profile {
	t.Helper()
	p, err := profile.LoadProfile("../../profiles/cisco-ios.yaml")
	if err != nil {
		t.Fatal(err)
	}
	return p
}

const fakeRunningConfig = "Building configuration...\n\n" +
	"Current configuration : 120 bytes\n!\nhostname sw1\n!\n" +
	"interface Loopback0\n ip address 192.0.2.1 255.255.255.255\n!\nend\n"

// devOpts changes how the fake device behaves for one test.
type devOpts struct {
	enableSecret string
	pager        bool // send --More-- and wait for a space
	truncate     bool // stop before the final "end" line
	silent       bool // never answer
}

func newFakeServerConfig(t *testing.T) *ssh.ServerConfig {
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
	return cfg
}

func startShellDevice(t *testing.T, o devOpts) string {
	t.Helper()
	cfg := newFakeServerConfig(t)
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
			go serveShell(nc, cfg, o)
		}
	}()
	return ln.Addr().String()
}

func serveShell(nc net.Conn, cfg *ssh.ServerConfig, o devOpts) {
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
		go handleShell(ch, creqs, o)
	}
}

// handleShell behaves like an IOS terminal: silent until a newline arrives,
// then a prompt after each command.
func handleShell(ch ssh.Channel, reqs <-chan *ssh.Request, o devOpts) {
	defer ch.Close()
	go func() {
		for r := range reqs {
			r.Reply(r.Type == "pty-req" || r.Type == "shell", nil)
		}
	}()
	if o.silent {
		io.Copy(io.Discard, ch)
		return
	}

	keys := make(chan byte, 4096)
	go func() {
		defer close(keys)
		b := make([]byte, 256)
		for {
			n, err := ch.Read(b)
			for _, c := range b[:n] {
				keys <- c
			}
			if err != nil {
				return
			}
		}
	}()

	prompt := "sw1>"
	say := func(s string) { io.WriteString(ch, s) }

	for {
		line, ok := readLine(keys)
		if !ok {
			return
		}
		switch {
		case line == "" || strings.HasPrefix(line, "terminal "):
			say(prompt)
		case line == "enable":
			say("Password: ")
			pw, ok := readLine(keys)
			if !ok {
				return
			}
			if pw == o.enableSecret {
				prompt = "sw1#"
				say("\r\n" + prompt)
			} else {
				say("\r\n% Access denied\r\n" + prompt)
			}
		case line == "show running-config":
			lines := strings.SplitAfter(fakeRunningConfig, "\n")
			if o.truncate {
				lines = lines[:len(lines)-2]
			}
			for i, l := range lines {
				if l == "" {
					continue
				}
				say(strings.TrimSuffix(l, "\n") + "\r\n")
				if o.pager && i == 6 {
					say(" --More-- ")
					if _, ok := readByte(keys); !ok {
						return
					}
					say("\r")
				}
			}
			say(prompt)
		case line == "exit":
			return
		default:
			say("% Unknown command\r\n" + prompt)
		}
	}
}

func readLine(keys <-chan byte) (string, bool) {
	var b strings.Builder
	for {
		c, ok := <-keys
		if !ok {
			return "", false
		}
		if c == '\n' {
			return strings.TrimRight(b.String(), "\r"), true
		}
		b.WriteByte(c)
	}
}

func readByte(keys <-chan byte) (byte, bool) {
	c, ok := <-keys
	return c, ok
}

func openTestShell(t *testing.T, o devOpts) *Shell {
	t.Helper()
	addr := startShellDevice(t, o)
	pin := filepath.Join(t.TempDir(), "fake.pin")
	client, err := Dial(addr, LegacyClientConfig("ro", "secret", PinnedHostKey(pin)))
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	t.Cleanup(func() { client.Close() })
	sh, err := OpenShell(client, testProfile(t))
	if err != nil {
		t.Fatalf("open shell: %v", err)
	}
	t.Cleanup(func() { sh.Close() })
	return sh
}

func TestEnableWithCorrectSecret(t *testing.T) {
	sh := openTestShell(t, devOpts{enableSecret: "enablesecret"})
	if _, err := sh.Sync(5 * time.Second); err != nil {
		t.Fatal(err)
	}
	if err := sh.Enable("enablesecret", 5*time.Second); err != nil {
		t.Fatalf("enable: %v", err)
	}
	out, err := sh.Run("show running-config", 5*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if !LooksComplete(out) {
		t.Fatalf("config not complete: %q", out)
	}
}

func TestEnableWrongSecretFailsOnce(t *testing.T) {
	sh := openTestShell(t, devOpts{enableSecret: "enablesecret"})
	if _, err := sh.Sync(5 * time.Second); err != nil {
		t.Fatal(err)
	}
	err := sh.Enable("wrong", 5*time.Second)
	if !errors.Is(err, ErrEnableFailed) {
		t.Fatalf("got %v, want ErrEnableFailed", err)
	}
}

func TestPagerIsAnsweredAndRemoved(t *testing.T) {
	sh := openTestShell(t, devOpts{enableSecret: "enablesecret", pager: true})
	if _, err := sh.Sync(5 * time.Second); err != nil {
		t.Fatal(err)
	}
	if err := sh.Enable("enablesecret", 5*time.Second); err != nil {
		t.Fatal(err)
	}
	out, err := sh.Run("show running-config", 5*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "--More--") {
		t.Fatal("pager marker left in output")
	}
	if !LooksComplete(out) {
		t.Fatalf("config incomplete after pager: %q", out)
	}
}

func TestTruncatedConfigIsIncomplete(t *testing.T) {
	sh := openTestShell(t, devOpts{enableSecret: "enablesecret", truncate: true})
	if _, err := sh.Sync(5 * time.Second); err != nil {
		t.Fatal(err)
	}
	if err := sh.Enable("enablesecret", 5*time.Second); err != nil {
		t.Fatal(err)
	}
	out, err := sh.Run("show running-config", 5*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if LooksComplete(out) {
		t.Fatal("truncated config accepted as complete")
	}
}

func TestSilentDeviceTimesOut(t *testing.T) {
	sh := openTestShell(t, devOpts{silent: true})
	start := time.Now()
	_, err := sh.Sync(2 * time.Second)
	if !errors.Is(err, ErrTimeout) {
		t.Fatalf("got %v, want ErrTimeout", err)
	}
	if time.Since(start) > 10*time.Second {
		t.Fatal("timeout took too long")
	}
}