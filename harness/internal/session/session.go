// Package session runs one model session with one prompt and answers its
// reply. Two commands use it: coldread hands a session the design and no
// tools, so that it has the document and nothing else; reverse hands one the
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
	// Verdict is the prefix of the line a cold-read session must end its reply with.
	Verdict string `yaml:"verdict"`
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
type Runner func(model, prompt string) (string, error)

// Claude runs a session through the claude command-line tool in print mode
// with no persisted session: the prompt on stdin, the reply on stdout. tools
// are the built-in tools the session may use, and none means none; dir is
// where it runs, which is what a session with tools can read.
func Claude(bin, dir string, tools ...string) Runner {
	return func(model, prompt string) (string, error) {
		cmd := exec.Command(bin, Args(model, tools)...)
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
// tool. Not --bare: that mode skips the stored login. No spending cap is
// ever passed: a session stopped short of the end of a large design or a
// large repository has answered nothing.
func Args(model string, tools []string) []string {
	return []string{"-p", "--tools", strings.Join(tools, ","), "--no-session-persistence", "--model", model}
}
