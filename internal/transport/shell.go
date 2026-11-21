package transport

import (
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"

	"netvault/internal/profile"
)

var (
	// ErrTimeout is returned when the device does not produce the expected text in time.
	ErrTimeout = errors.New("timed out waiting for device output")
	// ErrEnableFailed is returned when the enable secret is rejected.
	ErrEnableFailed = errors.New("enable failed: privileged prompt not reached")
)

// Shell is an interactive PTY session, the same as a person logged in at a
// terminal. Prompt, pager, and enable behavior come from the brand profile.
type Shell struct {
	sess      *ssh.Session
	in        io.WriteCloser
	pw        *io.PipeWriter
	chunks    chan []byte
	buf       strings.Builder
	userExec  *regexp.Regexp
	privExec  *regexp.Regexp
	passRe    *regexp.Regexp
	pagerRe   *regexp.Regexp
	pagerText string
	pagerAns  string
	enableCmd string
}

// OpenShell starts an interactive shell with a wide PTY and no local echo.
func OpenShell(client *ssh.Client, p *profile.Profile) (*Shell, error) {
	userRe, err := regexp.Compile(p.Prompt.UserExec)
	if err != nil {
		return nil, fmt.Errorf("prompt.user_exec: %w", err)
	}
	privRe, err := regexp.Compile(p.Prompt.Privileged)
	if err != nil {
		return nil, fmt.Errorf("prompt.privileged: %w", err)
	}
	passRe, err := regexp.Compile(p.Prompt.Password)
	if err != nil {
		return nil, fmt.Errorf("prompt.password: %w", err)
	}
	// Remove the pager marker and the escape sequences used to erase it.
	pagerRe := regexp.MustCompile(`\s*` + regexp.QuoteMeta(p.Prompt.Pager) + `\s*|\x08+|\x1b\[[0-9;]*[A-Za-z]`)

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

	s := &Shell{
		sess:      sess,
		in:        in,
		pw:        pw,
		chunks:    make(chan []byte, 64),
		userExec:  userRe,
		privExec:  privRe,
		passRe:    passRe,
		pagerRe:   pagerRe,
		pagerText: p.Prompt.Pager,
		pagerAns:  p.Prompt.PagerAnswer,
		enableCmd: p.Enable.Command,
	}
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

// isPrompt reports whether line is a user-exec or privileged prompt.
func (s *Shell) isPrompt(line string) bool {
	return s.userExec.MatchString(line) || s.privExec.MatchString(line)
}

// waitFor reads device output until done reports true for the text received
// so far. A pager marker is answered and removed from the text.
func (s *Shell) waitFor(done func(text string) bool, timeout time.Duration) (string, error) {
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	for {
		text := s.buf.String()
		if strings.HasSuffix(strings.TrimRight(lastLine(text), " "), s.pagerText) {
			s.buf.Reset()
			s.buf.WriteString(s.pagerRe.ReplaceAllString(text, ""))
			if _, err := io.WriteString(s.in, s.pagerAns); err != nil {
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
		return strings.TrimSpace(t) != "" && s.isPrompt(lastLine(t))
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
	return s.cleanOutput(text), nil
}

// Enable raises privilege using the profile's enable command and the secret.
// A wrong secret returns ErrEnableFailed after one attempt.
func (s *Shell) Enable(secret string, timeout time.Duration) error {
	s.buf.Reset()
	if _, err := io.WriteString(s.in, s.enableCmd+"\n"); err != nil {
		return err
	}
	text, err := s.waitFor(func(t string) bool {
		l := lastLine(t)
		return s.passRe.MatchString(l) || s.isPrompt(l)
	}, timeout)
	if err != nil {
		return err
	}
	last := strings.TrimSpace(lastLine(text))
	if s.isPrompt(last) {
		if s.privExec.MatchString(last) {
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
	if !s.privExec.MatchString(strings.TrimSpace(lastLine(text))) {
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
func (s *Shell) cleanOutput(text string) string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "")
	if i := strings.LastIndex(text, "\n"); i >= 0 && s.isPrompt(text[i+1:]) {
		return text[:i+1]
	}
	return text
}