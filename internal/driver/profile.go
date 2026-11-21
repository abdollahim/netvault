// Package driver loads brand profiles and runs commands through a shell
// session according to them.
package driver

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"regexp"
	"time"

	"gopkg.in/yaml.v3"
)

// Profile describes how one brand's command line behaves. It contains no site
// data, so committed profiles can be published.
type Profile struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	Transport   struct {
		KeyExchanges      []string `yaml:"key_exchanges"`
		Ciphers           []string `yaml:"ciphers"`
		MACs              []string `yaml:"macs"`
		HostKeyAlgorithms []string `yaml:"host_key_algorithms"`
	} `yaml:"transport"`
	Prompt struct {
		UserExec    string `yaml:"user_exec"`
		Privileged  string `yaml:"privileged"`
		Password    string `yaml:"password"`
		Pager       string `yaml:"pager"`
		PagerAnswer string `yaml:"pager_answer"`
	} `yaml:"prompt"`
	Setup  []string `yaml:"setup"`
	Enable struct {
		Command string `yaml:"command"`
	} `yaml:"enable"`
	Timeouts struct {
		ConnectSeconds int `yaml:"connect_seconds"`
		PromptSeconds  int `yaml:"prompt_seconds"`
		CommandSeconds int `yaml:"command_seconds"`
	} `yaml:"timeouts"`
}

// LoadProfile reads and validates a profile. Unknown fields are rejected, so a
// misspelled key fails at load time instead of being silently ignored.
func LoadProfile(path string) (*Profile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read profile: %w", err)
	}
	var p Profile
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err := dec.Decode(&p); err != nil {
		return nil, fmt.Errorf("parse profile %s: %w", path, err)
	}
	if err := p.Validate(); err != nil {
		return nil, fmt.Errorf("profile %s: %w", path, err)
	}
	return &p, nil
}

// Validate checks required fields and compiles every pattern, naming the
// field that is wrong.
func (p *Profile) Validate() error {
	if p.Name == "" {
		return errors.New("name is required")
	}
	if len(p.Transport.KeyExchanges) == 0 || len(p.Transport.Ciphers) == 0 {
		return errors.New("transport.key_exchanges and transport.ciphers are required")
	}
	if len(p.Transport.HostKeyAlgorithms) == 0 {
		return errors.New("transport.host_key_algorithms is required")
	}
	patterns := []struct{ field, value string }{
		{"prompt.user_exec", p.Prompt.UserExec},
		{"prompt.privileged", p.Prompt.Privileged},
		{"prompt.password", p.Prompt.Password},
	}
	for _, pat := range patterns {
		if pat.value == "" {
			return fmt.Errorf("%s is required", pat.field)
		}
		if _, err := regexp.Compile(pat.value); err != nil {
			return fmt.Errorf("%s is not a valid pattern: %w", pat.field, err)
		}
	}
	if p.Prompt.Pager == "" || p.Prompt.PagerAnswer == "" {
		return errors.New("prompt.pager and prompt.pager_answer are required")
	}
	if p.Enable.Command == "" {
		return errors.New("enable.command is required")
	}
	if p.Timeouts.ConnectSeconds <= 0 || p.Timeouts.PromptSeconds <= 0 || p.Timeouts.CommandSeconds <= 0 {
		return errors.New("all timeouts must be positive")
	}
	return nil
}

// ConnectTimeout returns the connection timeout as a duration.
func (p *Profile) ConnectTimeout() time.Duration {
	return time.Duration(p.Timeouts.ConnectSeconds) * time.Second
}

// PromptTimeout returns the time to wait for a prompt.
func (p *Profile) PromptTimeout() time.Duration {
	return time.Duration(p.Timeouts.PromptSeconds) * time.Second
}

// CommandTimeout returns the time allowed for one command's output.
func (p *Profile) CommandTimeout() time.Duration {
	return time.Duration(p.Timeouts.CommandSeconds) * time.Second
}