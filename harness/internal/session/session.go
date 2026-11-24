// Package session runs one model session with one prompt and answers its
// reply. Both animals use it: goldfish hands a session the design and no
// tools, so that it has the document and nothing else; elephant hands one the
// repository to read, and nothing to write with.
package session

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"gopkg.in/yaml.v3"
)

// Settings are a command's values, read from a YAML file beside its prompts.
type Settings struct {
	// Model runs the sessions and is recorded with what each said.
	Model string `yaml:"model"`
	// Verdict is the prefix of the line a goldfish must end its reply with.
	Verdict string `yaml:"verdict"`
	// MaxBudgetUSD is the most one session may spend. 0 means no cap.
	MaxBudgetUSD float64 `yaml:"max-budget-usd"`
}

// Load reads a settings file. A model is always required.
func Load(path string) (Settings, error) {
	var s Settings
	b, err := os.ReadFile(path)
	if err != nil {
		return s, err
	}
	if err := yaml.Unmarshal(b, &s); err != nil {
		return s, fmt.Errorf("%s: %w", path, err)
	}
	if strings.TrimSpace(s.Model) == "" {
		return s, fmt.Errorf("%s: model is empty", path)
	}
	return s, nil
}

// Runner runs one session with one prompt and answers the reply. The real one
// is the claude command-line tool; the tests substitute their own.
type Runner func(model string, budget float64, prompt string) (string, error)

// Claude runs a session through the claude command-line tool in print mode
// with no persisted session: the prompt on stdin, the reply on stdout. tools
// are the built-in tools the session may use, and none means none; dir is
// where it runs, which is what a session with tools can read.
func Claude(bin, dir string, tools ...string) Runner {
	return func(model string, budget float64, prompt string) (string, error) {
		cmd := exec.Command(bin, Args(model, budget, tools)...)
		cmd.Dir = dir
		cmd.Stdin = strings.NewReader(prompt)
		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		if err := cmd.Run(); err != nil {
			return stdout.String(), fmt.Errorf("%s: %w: %s", bin, err, strings.TrimSpace(stderr.String()))
		}
		return stdout.String(), nil
	}
}

// Args are the arguments a session is started with. --tools is always given,
// empty when the session has none: left out, the tool's default is every
// tool. Not --bare: that mode skips the stored login.
func Args(model string, budget float64, tools []string) []string {
	args := []string{"-p", "--tools", strings.Join(tools, ","), "--no-session-persistence", "--model", model}
	if budget > 0 {
		args = append(args, "--max-budget-usd", fmt.Sprintf("%g", budget))
	}
	return args
}
