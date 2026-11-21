// Command spike is a throwaway tool. It runs one command on one device in an
// interactive shell and reports output size, completeness, and timing.
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"internal/profile"
	"internal/transport"
	"golang.org/x/crypto/ssh"
	"golang.org/x/term"
)

func main() {
	addr := flag.String("addr", "", "device address host:port (required)")
	user := flag.String("user", "", "read-only account name (required)")
	cmd := flag.String("cmd", "show running-config", "command to run")
	pin := flag.String("pin", "", "host key pin file, kept outside the repo (required)")
	flag.Parse()

	if *addr == "" || *user == "" || *pin == "" {
		fail("-addr, -user, and -pin are required")
	}

	if *user == "" {
		fail("-user is required")
	}
	pw := readSecret("password for " + *user + ": ")

	client, err := transport.Dial(*addr,
		transport.LegacyClientConfig(*user, string(pw), transport.PinnedHostKey(*pin)))
	if err != nil {
		fail(err.Error())
	}
	defer client.Close()

	start := time.Now()
	out, err := runInShell(client, *cmd)
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

// runInShell logs in to the interactive shell, raises privilege if needed,
// disables paging, and runs the command.
func runInShell(client *ssh.Client, cmd string) (string, error) {
	sh, err := transport.OpenShell(client)
	if err != nil {
		return "", err
	}
	defer sh.Close()

	prof, err := profile.LoadProfile("profiles/cisco-ios.yaml")
	if err != nil {
		fail(err.Error())
	}
	if strings.HasSuffix(prompt, ">") {
		secret := readSecret("enable secret: ")
		if err := sh.Enable(string(secret), 30*time.Second); err != nil {
			return "", err
		}
	}
	if _, err := sh.Run("terminal length 0", 30*time.Second); err != nil {
		return "", fmt.Errorf("terminal length: %w", err)
	}
	if _, err := sh.Run("terminal width 0", 30*time.Second); err != nil {
		return "", fmt.Errorf("terminal width: %w", err)
	}
	return sh.Run(cmd, 10*time.Minute)
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