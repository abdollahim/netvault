package transport

import (
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

var (
	// promptLine matches a bare IOS prompt such as "host#" or "host>".
	promptLine = regexp.MustCompile(`^[\w.\-()/]+[>#]\s*$`)
	// pagerStrip removes the pager marker and the escape sequences used to erase it.
	pagerStrip = regexp.MustCompile(`\s*--More--\s*|\x08+|\x1b\[[0-9;]*[A-Za-z]`)

	// ErrTimeout is returned when the device does not produce the expected text in time.
	ErrTimeout = errors.New("timed out waiting for device output")
	// ErrEnableFailed is returned when the enable secret is rejected.
	ErrEnableFailed = errors.New("enable failed: privilege level 15 not reached")
)

// Shell is an interactive PTY session, the same as a person logged in at a
// terminal. It is used when exec-channel output comes back incomplete.
type Shell struct {
	sess   *ssh.Session
	in     io.WriteCloser
	pw     *io.PipeWriter
	chunks chan []byte
	buf    strings.Builder
}

// OpenShell starts an interactive shell with a wide PTY and no local echo.
func OpenShell(client *ssh.Client) (*Shell, error) {
	sess, err := client.NewSession()
	if err != nil {
		return nil, fmt.Errorf("new session: %w", err)
	}
	modes := ssh.TerminalModes{
		ssh.ECHO:          0,
		ssh.TTY_OP_ISPEED: 9600,
		ssh.TTY_OP_OSPEED: 9600,
	}
	if err := sess.RequestPty("vt100", 200, 512, modes); err != nil {
		sess.Close()
		return nil, fmt.Errorf("request pty: %w", err)
	}
	in, err := sess.StdinPipe()
	if err != nil {
		sess.Close()
		return nil, fmt.Errorf("stdin: %w", err)
	}
	pr, pw := io.Pipe()
	sess.Stdout = pw
	sess.Stderr = pw
	if err := sess.Shell(); err != nil {
		sess.Close()
		pw.Close()
		return nil, fmt.Errorf("start shell: %w", err)
	}
	s := &Shell{sess: sess, in: in, pw: pw, chunks: make(chan []byte, 64)}
	go s.pump(pr)
	return s, nil
}

// pump copies device output into the chunks channel until the stream ends.
func (s *Shell) pump(r io.Reader) {
	defer close(s.chunks)
	buf := make([]byte, 4096)
	for {
		n, err := r.Read(buf)
		if n > 0 {
			c := make([]byte, n)
			copy(c, buf[:n])
			s.chunks <- c
		}
		if err != nil {
			return
		}
	}
}

// waitFor reads device output until done reports true for the text received
// so far. A pager marker is answered with a space and removed from the text.
func (s *Shell) waitFor(done func(text string) bool, timeout time.Duration) (string, error) {
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	for {
		text := s.buf.String()
		if strings.HasSuffix(strings.TrimRight(lastLine(text), " "), "--More--") {
			s.buf.Reset()
			s.buf.WriteString(pagerStrip.ReplaceAllString(text, ""))
			if _, err := io.WriteString(s.in, " "); err != nil {
				return "", err
			}
			continue
		}
		if done(text) {
			s.buf.Reset()
			return text, nil
		}
		select {
		case c, ok := <-s.chunks:
			if !ok {
				return "", io.ErrUnexpectedEOF
			}
			s.buf.Write(c)
		case <-timer.C:
			return "", ErrTimeout
		}
	}
}

// readPrompt reads until the last line of output is a prompt.
func (s *Shell) readPrompt(timeout time.Duration) (string, error) {
	return s.waitFor(func(t string) bool {
		return strings.TrimSpace(t) != "" && promptLine.MatchString(lastLine(t))
	}, timeout)
}

// Sync sends an empty line and returns the prompt that comes back. It also
// consumes the login banner, so later commands start from a clean prompt.
func (s *Shell) Sync(timeout time.Duration) (string, error) {
	s.buf.Reset()
	if _, err := io.WriteString(s.in, "\n"); err != nil {
		return "", err
	}
	text, err := s.readPrompt(timeout)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(lastLine(text)), nil
}

// Run sends one command and returns its output, without the trailing prompt.
func (s *Shell) Run(cmd string, timeout time.Duration) (string, error) {
	s.buf.Reset()
	if _, err := io.WriteString(s.in, cmd+"\n"); err != nil {
		return "", err
	}
	text, err := s.readPrompt(timeout)
	if err != nil {
		return "", err
	}
	return cleanOutput(text), nil
}

// Enable raises privilege to level 15 using the enable secret.
func (s *Shell) Enable(secret string, timeout time.Duration) error {
	s.buf.Reset()
	if _, err := io.WriteString(s.in, "enable\n"); err != nil {
		return err
	}
	text, err := s.waitFor(func(t string) bool {
		l := lastLine(t)
		return strings.Contains(strings.ToLower(l), "password") || promptLine.MatchString(l)
	}, timeout)
	if err != nil {
		return err
	}
	last := strings.TrimSpace(lastLine(text))
	if promptLine.MatchString(last) {
		if strings.HasSuffix(last, "#") {
			return nil // enabled without a secret
		}
		return ErrEnableFailed
	}
	s.buf.Reset()
	if _, err := io.WriteString(s.in, secret+"\n"); err != nil {
		return err
	}
	text, err = s.readPrompt(timeout)
	if err != nil {
		return err
	}
	if !strings.HasSuffix(strings.TrimSpace(lastLine(text)), "#") {
		return ErrEnableFailed
	}
	return nil
}

// Close ends the shell and the underlying session.
func (s *Shell) Close() error {
	io.WriteString(s.in, "exit\n")
	s.in.Close()
	s.pw.Close()
	return s.sess.Close()
}

// LooksComplete reports whether text is a whole IOS running configuration:
// it has the header and its last line is "end".
func LooksComplete(text string) bool {
	trimmed := strings.TrimSpace(text)
	return strings.Contains(trimmed, "Current configuration") && lastLine(trimmed) == "end"
}

// lastLine returns the text after the final newline.
func lastLine(s string) string {
	if i := strings.LastIndex(s, "\n"); i >= 0 {
		return s[i+1:]
	}
	return s
}

// cleanOutput normalizes line endings and drops the trailing prompt line.
func cleanOutput(text string) string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "")
	if i := strings.LastIndex(text, "\n"); i >= 0 && promptLine.MatchString(text[i+1:]) {
		return text[:i+1]
	}
	return text
}