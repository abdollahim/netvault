// Command spike is a throwaway Phase 1 tool. It connects to one device with
// legacy SSH settings, runs one command, and reports output size and timing.
package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/abdollahim/netvault/internal/transport"
	"golang.org/x/term"
)

func main() {
	addr := flag.String("addr", "REDACTED-IP:22", "device address host:port")
	user := flag.String("user", "", "read-only account name")
	cmd := flag.String("cmd", "show version", "command to run")
	pin := flag.String("pin", "spike-REDACTED-IP.pin", "host key pin file")
	flag.Parse()

	if *user == "" {
		fail("-user is required")
	}

	fmt.Fprintf(os.Stderr, "password for %s: ", *user)
	pw, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Fprintln(os.Stderr)
	if err != nil {
		fail(err.Error())
	}

	start := time.Now()
	client, err := transport.Dial(*addr,
		transport.LegacyClientConfig(*user, string(pw), transport.PinnedHostKey(*pin)))
	if err != nil {
		fail(err.Error())
	}
	defer client.Close()

	sess, err := client.NewSession()
	if err != nil {
		fail(err.Error())
	}
	defer sess.Close()

	out, err := sess.Output(*cmd)
	if err != nil {
		fail(err.Error())
	}
	os.Stdout.Write(out)
	fmt.Fprintf(os.Stderr, "\nbytes=%d elapsed=%s\n", len(out), time.Since(start).Round(time.Millisecond))
}

func fail(msg string) {
	fmt.Fprintln(os.Stderr, "spike:", msg)
	os.Exit(1)
}