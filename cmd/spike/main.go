// Command spike is a throwaway tool. It runs one command on one device in an
// interactive shell and reports output size, completeness, and timing.
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"netvault/internal/profile"
	"netvault/internal/transport"
	"golang.org/x/crypto/ssh"
	"golang.org/x/term"
)

func main() {
	addr := flag.String("addr", "", "device address host:port (required)")
	user := flag.String("user", "", "read-only account name (required)")
	cmd := flag.String("cmd", "show running-config", "command to run")
	pin := flag.String("pin", "", "host key pin file, kept outside the repo (required)")
	profPath := flag.String("profile", "profiles/cisco-ios.yaml", "brand profile file")
	flag.Parse()

	if *addr == "" || *user == "" || *pin == "" {
		fail("-addr, -user, and -pin are required")
	}

	prof, err := profile.LoadProfile(*profPath)
	if err != nil {
		fail(err.Error())
	}

	pw := readSecret("password for " + *user + ": ")

	client, err := transport.Dial(*addr,
		transport.ClientConfigFromProfile(prof, *user, string(pw), transport.PinnedHostKey(*pin)))
	if err != nil {
		fail(err.Error())
	}
	defer client.Close()

	start := time.Now()
	out, err := runInShell(client, prof, *cmd)
	if err != nil {
		fail(err.Error())
	}
	os.Stdout.WriteString(out)

	complete := "n/a"
	if strings.Contains(*cmd, "running-config") {
		complete = fmt.Sprint(transport.LooksComplete(out))
	}
	fmt.Fprintf(os.Stderr, "\nbytes=%d complete=%s elapsed=%s\n",
		len(out), complete, time.Since(start).Round(time.Millisecond))
}

// runInShell logs in to the interactive shell, raises privilege when the
// profile's privileged prompt is required, runs the setup commands, and runs
// the command.
func runInShell(client *ssh.Client, prof *profile.Profile, cmd string) (string, error) {
	sh, err := transport.OpenShell(client, prof)
	if err != nil {
		return "", err
	}
	defer sh.Close()

	prompt, err := sh.Sync(prof.PromptTimeout())
	if err != nil {
		return "", fmt.Errorf("login prompt: %w", err)
	}
	if strings.HasSuffix(prompt, ">") {
		secret := readSecret("enable secret: ")
		if err := sh.Enable(string(secret), prof.PromptTimeout()); err != nil {
			return "", err
		}
	}
	for _, setup := range prof.Setup {
		if _, err := sh.Run(setup, prof.PromptTimeout()); err != nil {
			return "", fmt.Errorf("setup %q: %w", setup, err)
		}
	}
	return sh.Run(cmd, prof.CommandTimeout())
}

func readSecret(prompt string) []byte {
	fmt.Fprint(os.Stderr, prompt)
	b, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Fprintln(os.Stderr)
	if err != nil {
		fail(err.Error())
	}
	return b
}

func fail(msg string) {
	fmt.Fprintln(os.Stderr, "spike:", msg)
	os.Exit(1)
}