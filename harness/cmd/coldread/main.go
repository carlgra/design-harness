// Command coldread asks two sessions with no memory whether a design is
// ready, each handed design/DESIGN.md and one question. Comprehension: say
// what the repository is for and how it works. Readiness: for the change at
// hand, say whether the design is enough to implement it in one pass.
//
// Run it by hand when a design changes materially; it is not a gate. Each
// session is the claude command-line tool in print mode with no tools and no
// persisted session, so that it has the design and nothing else. The model,
// and the verdict line are in harness/coldread/coldread.yaml; the
// two prompts are beside it.
//
//	go -C harness run ./cmd/coldread -change "what the pull request does"
//	go -C harness run ./cmd/coldread -change-file note.md -out coldread.md
//	go -C harness run ./cmd/coldread -only comprehension
//
// It prints each verdict with the session's reasons and writes them as
// markdown to -out, to paste into the pull request. A verdict is advice for
// the person who merges; the person decides.
package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"harness/internal/design"
	"harness/internal/session"
)

// testNames are the two, in the order they are run and reported. Each has a
// prompt file of the same name beside the settings.
var testNames = []string{"comprehension", "readiness"}

type test struct {
	Name   string
	Prompt string
}

type verdict struct {
	Test  string
	Model string
	Pass  bool
	Reply string
	Err   error
}

func main() {
	rootFlag := flag.String("root", "", "repository root (default: walk up from the working directory)")
	changeFlag := flag.String("change", "", "what the pull request does, for the readiness test")
	changeFile := flag.String("change-file", "", "read the change from this file instead, a design note for instance")
	outFlag := flag.String("out", "", "write the verdicts as markdown to this path, for the pull request")
	claudeFlag := flag.String("claude", "claude", "the claude command-line tool")
	onlyFlag := flag.String("only", "", "run one test only: comprehension or readiness")
	flag.Parse()
	design.ForgetGitEnv()

	root, err := design.FindRoot(*rootFlag)
	if err != nil {
		fatal(err)
	}
	repo, err := design.Load(root)
	if err != nil {
		fatal(err)
	}
	harness, err := design.FindHarness(root)
	if err != nil {
		fatal(err)
	}
	headPath := filepath.Join(root, filepath.FromSlash(design.DocPath))
	head, err := os.ReadFile(headPath)
	if err != nil {
		fatal(err)
	}

	s, err := loadSettings(harness)
	if err != nil {
		fatal(err)
	}
	tests, err := loadTests(harness, *onlyFlag)
	if err != nil {
		fatal(err)
	}

	change := *changeFlag
	if *changeFile != "" {
		p, _ := design.Abs(*changeFile)
		b, err := os.ReadFile(p)
		if err != nil {
			fatal(err)
		}
		change = string(b)
	}
	if asksReadiness(tests) && strings.TrimSpace(change) == "" {
		fatal(errors.New("the readiness test needs the change: -change \"...\" or -change-file <path>; or ask comprehension alone with -only comprehension"))
	}

	fmt.Printf("coldread: %s: %d session(s) with no memory, model %s, reading %s\n", repo.Name, len(tests), s.Model, design.Rel(root, headPath))
	verdicts := run(tests, string(head), change, s, session.Claude(*claudeFlag, os.TempDir()))
	failed := false
	for _, v := range verdicts {
		fmt.Println()
		fmt.Println(headline(v))
		fmt.Println(indent(reasons(v)))
		if !v.Pass {
			failed = true
		}
	}
	if *outFlag != "" {
		p, _ := design.Abs(*outFlag)
		if err := os.WriteFile(p, []byte(markdown(repo.Name, verdicts)), 0o644); err != nil {
			fatal(err)
		}
	}
	fmt.Println()
	if failed {
		fmt.Println("coldread: a session with no memory could not explain or build from this design; the reasons are above. Read them before merging.")
		os.Exit(1)
	}
	fmt.Println("coldread: ready. Every session passed.")
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "coldread:", err)
	os.Exit(2)
}

func asksReadiness(tests []test) bool {
	for _, t := range tests {
		if t.Name == "readiness" {
			return true
		}
	}
	return false
}

// loadSettings reads the settings beside the prompts.
func loadSettings(harness string) (session.Settings, error) {
	p := filepath.Join(harness, "coldread", "coldread.yaml")
	s, err := session.Load(p)
	if err != nil {
		return s, err
	}
	if strings.TrimSpace(s.Verdict) == "" {
		return s, fmt.Errorf("%s: verdict is empty", p)
	}
	return s, nil
}

// loadTests reads the two prompts, or the one asked for.
func loadTests(harness, only string) ([]test, error) {
	var tests []test
	for _, name := range testNames {
		if only != "" && only != name {
			continue
		}
		b, err := os.ReadFile(filepath.Join(harness, "coldread", name+".md"))
		if err != nil {
			return nil, err
		}
		tests = append(tests, test{Name: name, Prompt: string(b)})
	}
	if len(tests) == 0 {
		return nil, fmt.Errorf("-only %q is not one of %s", only, strings.Join(testNames, ", "))
	}
	return tests, nil
}

// compose is the whole of what a session is given: its prompt, the design,
// and for the readiness test the change. Nothing else is ever added here; the
// test pins that.
func compose(t test, head, change string) string {
	b := &strings.Builder{}
	b.WriteString(strings.TrimRight(t.Prompt, "\n"))
	b.WriteString("\n\n")
	b.WriteString(strings.TrimSpace(head))
	b.WriteString("\n")
	if t.Name == "readiness" {
		b.WriteString("\n---\n\n")
		b.WriteString(strings.TrimSpace(change))
		b.WriteString("\n")
	}
	return b.String()
}

// run asks each test of its own fresh session and reads the verdicts.
func run(tests []test, head, change string, s session.Settings, r session.Runner) []verdict {
	var out []verdict
	for _, t := range tests {
		v := verdict{Test: t.Name, Model: s.Model}
		reply, err := r(s.Model, compose(t, head, change))
		v.Reply = reply
		if err != nil {
			v.Err = err
		} else if pass, found := parseVerdict(reply, s.Verdict); !found {
			v.Err = fmt.Errorf("the reply ends with no %q line; nothing else counts as a verdict", s.Verdict)
		} else {
			v.Pass = pass
		}
		out = append(out, v)
	}
	return out
}

// parseVerdict reads the verdict from the last non-empty line of a reply.
// It answers (pass, found): found is false when that line does not start
// with the prefix or says neither PASS nor FAIL. The script judges nothing;
// a reply that does not end in the form is not a pass.
func parseVerdict(reply, prefix string) (pass, found bool) {
	lines := strings.Split(strings.TrimSpace(reply), "\n")
	last := strings.TrimSpace(lines[len(lines)-1])
	if !strings.HasPrefix(last, prefix) {
		return false, false
	}
	switch strings.ToUpper(strings.TrimSpace(strings.TrimPrefix(last, prefix))) {
	case "PASS":
		return true, true
	case "FAIL":
		return false, true
	}
	return false, false
}

func headline(v verdict) string {
	switch {
	case v.Err != nil:
		return fmt.Sprintf("%s: FAIL (%v)", v.Test, v.Err)
	case v.Pass:
		return v.Test + ": PASS"
	}
	return v.Test + ": FAIL"
}

func reasons(v verdict) string {
	r := strings.TrimSpace(v.Reply)
	if r == "" {
		return "(no reply)"
	}
	return r
}

func indent(s string) string {
	return "  " + strings.ReplaceAll(s, "\n", "\n  ")
}

// markdown is what goes on the pull request: each verdict with its reasons.
func markdown(repo string, verdicts []verdict) string {
	b := &strings.Builder{}
	fmt.Fprintf(b, "## Cold read: %s\n\n", repo)
	b.WriteString("Sessions with no memory, each handed `design/DESIGN.md` and one question. Advice for whoever merges, not a gate.\n\n")
	for _, v := range verdicts {
		fmt.Fprintf(b, "<details>\n<summary><b>%s</b> · model %s</summary>\n\n%s\n\n</details>\n\n", headline(v), v.Model, reasons(v))
	}
	return b.String()
}
