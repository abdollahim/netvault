package transport

import (
	"crypto/ed25519"
	"crypto/rand"
	"path/filepath"
	"testing"

	"golang.org/x/crypto/ssh"
)

func TestPinnedHostKeyRefusesChangedKey(t *testing.T) {
	pin := filepath.Join(t.TempDir(), "device.pin")
	cb := PinnedHostKey(pin)
	first := newTestKey(t)
	second := newTestKey(t)

	if err := cb("device", nil, first); err != nil {
		t.Fatalf("first use: %v", err)
	}
	if err := cb("device", nil, first); err != nil {
		t.Fatalf("same key: %v", err)
	}
	if err := cb("device", nil, second); err == nil {
		t.Fatal("changed key accepted")
	}
}

func newTestKey(t *testing.T) ssh.PublicKey {
	t.Helper()
	pub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	key, err := ssh.NewPublicKey(pub)
	if err != nil {
		t.Fatal(err)
	}
	return key
}