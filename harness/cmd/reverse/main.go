// Command reverse writes down what a repository that already exists has
// never said: its design. design/DESIGN.md is the one thing here that must
// not be forgotten; this command is how a repository whose
// design lives in its code and in people's heads gets one.
//
// It works in steps, each less mechanical than the last:
//
//	go -C harness run ./cmd/reverse survey     # what a machine can say: files, tests, gate, documents, commits
//	go -C harness run ./cmd/reverse skeleton   # write design/DESIGN.md with a spine that accounts for every file, and questions
//	go -C harness run ./cmd/reverse prompt     # print what a drafting session is handed, to give to an agent of your own
//	go -C harness run ./cmd/reverse draft      # one session reads the repository and drafts the design and the project's rules
//
// The survey decides nothing. The draft is a session that can read the
// repository and write nothing: this command writes what it replies, to
// design/DESIGN.md and to the project's section of CLAUDE.md, and only over a
// skeleton unless -force is given. What the session could not find in the
// repository it marks "(inferred)" or TODO(design), and a person answers
// each before the design is the design. The model is in
// harness/reverse/reverse.yaml; the prompt is beside it.
package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"harness/internal/design"
	"harness/internal/session"
)

const (
	// SkeletonMark is the line a skeleton carries and a design does not: a
	// draft may replace a document that has it.
	SkeletonMark = "<!-- design:skeleton -->"
	// DesignHeader and RulesHeader open the two parts of a drafting
	// session's reply.
	DesignHeader = "=== design/DESIGN.md ==="
	RulesHeader  = "=== CLAUDE.md ==="
	// ProjectBegin and ProjectEnd bracket the part of CLAUDE.md that is the
	// project's own, which a draft writes; everything outside is left alone.
	ProjectBegin = "<!-- design:project:begin -->"
	ProjectEnd   = "<!-- design:project:end -->"
)

func main() {
	if len(os.Args) < 2 {
		usage()
	}
	step := os.Args[1]
	fs := flag.NewFlagSet("reverse "+step, flag.ExitOnError)
	rootFlag := fs.String("root", "", "the project (default: the git repository the command was typed in)")
	forceFlag := fs.Bool("force", false, "replace a design/DESIGN.md that is no longer a skeleton")
	claudeFlag := fs.String("claude", "claude", "the claude command-line tool")
	fs.Parse(os.Args[2:])
	design.ForgetGitEnv()

	root, err := design.FindProject(*rootFlag)
	if err != nil {
		fatal(err)
	}
	harness, err := design.FindHarness(root)
	if err != nil {
		fatal(err)
	}
	repo, err := design.Walk(root)
	if err != nil {
		fatal(err)
	}
	parent, err := design.FindParent(root)
	if err != nil {
		fatal(err)
	}
	s := survey(repo, parent, recentSubjects(root, 200), time.Now().Format("2006-01-02"))
	template, err := os.ReadFile(filepath.Join(harness, "templates", "DESIGN.md"))
	if err != nil {
		fatal(err)
	}
	docPath := filepath.Join(root, filepath.FromSlash(design.DocPath))

	switch step {
	case "survey":
		fmt.Print(s.Markdown())

	case "skeleton":
		if _, err := os.Stat(docPath); err == nil {
			fatal(fmt.Errorf("%s already exists; a skeleton is only ever the first one", design.DocPath))
		}
		if err := write(docPath, s.Skeleton(string(template))); err != nil {
			fatal(err)
		}
		fmt.Printf("reverse: wrote %s: a spine that accounts for all %d files, and every section still a question\n", design.DocPath, s.Files)
		// Adopting the method is itself a change, so it has a note: the
		// first, unless the repository already keeps notes.
		changes := filepath.Join(root, "design", "changes")
		if existing, _ := filepath.Glob(filepath.Join(changes, "*.md")); len(existing) == 0 {
			adopt, err := os.ReadFile(filepath.Join(harness, "templates", "adopt.md"))
			if err != nil {
				fatal(err)
			}
			name := s.Date + "-adopt-the-design.md"
			if err := write(filepath.Join(changes, name), s.AdoptionNote(string(adopt))); err != nil {
				fatal(err)
			}
			fmt.Printf("reverse: wrote design/changes/%s\n", name)
		}
		fmt.Printf("reverse: next, draft it:  go -C %s run ./cmd/reverse draft\n", s.HarnessPath())

	case "where":
		// Where the repository sits, for a script to read: one key=value a
		// line. bootstrap uses it to write a CI workflow that checks out
		// what the checker will need.
		if parent != nil {
			fmt.Printf("root=%s\npath=%s\n", parent.Name, parent.Path)
		}
		if c := s.Contracts(); c != nil {
			fmt.Printf("contracts_path=%s\ncontracts_remote=%s\n", strings.Trim(c.Path, "/"), c.Remote)
		}

	case "prompt":
		p, err := prompt(harness, root, s, string(template))
		if err != nil {
			fatal(err)
		}
		fmt.Print(p)

	case "draft":
		if err := replaceable(docPath, *forceFlag); err != nil {
			fatal(err)
		}
		set, err := session.Load(filepath.Join(harness, "reverse", "reverse.yaml"))
		if err != nil {
			fatal(err)
		}
		p, err := prompt(harness, root, s, string(template))
		if err != nil {
			fatal(err)
		}
		fmt.Printf("reverse: %s: one session, model %s, reading %d files and writing none\n", s.Name, set.Model, s.Files)
		reply, err := session.Claude(*claudeFlag, root, "Read", "Grep", "Glob")(set.Model, p)
		if err != nil {
			fatal(err)
		}
		doc, rules, err := split(reply)
		if err != nil {
			// What a session said was paid for; it is kept, not lost.
			kept := filepath.Join(root, "design", "reverse-reply.md")
			if werr := write(kept, reply); werr == nil {
				err = fmt.Errorf("%w; the reply is kept in %s", err, design.Rel(root, kept))
			}
			fatal(err)
		}
		if err := write(docPath, doc); err != nil {
			fatal(err)
		}
		fmt.Printf("reverse: wrote %s\n", design.DocPath)
		if rules != "" {
			claude := filepath.Join(root, "CLAUDE.md")
			old, _ := os.ReadFile(claude)
			if err := write(claude, project(string(old), rules)); err != nil {
				fatal(err)
			}
			fmt.Println("reverse: wrote the project's section of CLAUDE.md")
		}
		fmt.Printf("reverse: %d place(s) are marked (inferred) and %d TODO(design): a person answers each.\n",
			strings.Count(doc, "(inferred)"), strings.Count(doc, "TODO(design)"))
		fmt.Println("reverse: next:")
		fmt.Printf("    go -C %s run ./cmd/check -validate=false       # is every file accounted for, does every named test exist\n", s.HarnessPath())
		fmt.Printf("    go -C %s run ./cmd/coldread -only comprehension # can a session with no memory explain the repository from it\n", s.HarnessPath())

	default:
		usage()
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: reverse survey|skeleton|prompt|draft|where [-root DIR] [-force] [-claude BIN]")
	os.Exit(2)
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "reverse:", err)
	os.Exit(2)
}

func write(p, content string) error {
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	return os.WriteFile(p, []byte(content), 0o644)
}

// replaceable refuses to draft over a design a person has taken on: one that
// exists and is no longer a skeleton. The design is the one thing that must
// not be forgotten, and a draft is a guess.
func replaceable(docPath string, force bool) error {
	b, err := os.ReadFile(docPath)
	if os.IsNotExist(err) || force {
		return nil
	}
	if err != nil {
		return err
	}
	if !strings.Contains(string(b), SkeletonMark) {
		return fmt.Errorf("%s is no longer a skeleton, so a draft would replace a design someone wrote; -force replaces it", design.DocPath)
	}
	return nil
}

// prompt is the whole of what a drafting session is given: the instructions,
// the survey, the shape to fill and, for a repository nested under a root,
// the root's design.
func prompt(harness, root string, s Survey, template string) (string, error) {
	instructions, err := os.ReadFile(filepath.Join(harness, "reverse", "reverse.md"))
	if err != nil {
		return "", err
	}
	b := &strings.Builder{}
	b.WriteString(strings.TrimRight(string(instructions), "\n"))
	b.WriteString("\n\n---\n\n")
	b.WriteString(strings.TrimSpace(s.Markdown()))
	b.WriteString("\n\n---\n\nThe shape to fill, with the spine as the survey gives it:\n\n````markdown\n")
	b.WriteString(strings.TrimSpace(strings.ReplaceAll(s.Skeleton(template), SkeletonMark+"\n\n", "")))
	b.WriteString("\n````\n")
	if p := s.Parent; p != nil {
		fmt.Fprintf(b, "\n---\n\nThe design of `%s`, the root this repository is nested under. What it decides binds here: cite it, and do not restate it.\n\n````markdown\n", p.Name)
		b.WriteString(strings.TrimSpace(p.Doc))
		b.WriteString("\n````\n")
	}
	return b.String(), nil
}

// split reads a drafting session's reply: the design after its header, and
// the project's rules after theirs. The design must be a document the checker
// can load: frontmatter that parses, with files: in it.
func split(reply string) (doc, rules string, err error) {
	reply = strings.ReplaceAll(reply, "\r\n", "\n")
	i := strings.Index(reply, DesignHeader)
	if i < 0 {
		return "", "", fmt.Errorf("the reply has no %q line", DesignHeader)
	}
	doc = reply[i+len(DesignHeader):]
	if j := strings.Index(doc, RulesHeader); j >= 0 {
		rules = strings.TrimSpace(unfence(doc[j+len(RulesHeader):]))
		doc = doc[:j]
	}
	doc = strings.TrimSpace(unfence(doc)) + "\n"
	fm, err := design.ParseFrontmatter(doc)
	if err != nil {
		return "", "", fmt.Errorf("the design in the reply: %w", err)
	}
	var d design.Design
	if err := yaml.Unmarshal([]byte(fm.YAML), &d); err != nil {
		return "", "", fmt.Errorf("the design in the reply: frontmatter: %w", err)
	}
	if len(d.Files) == 0 {
		return "", "", errors.New("the design in the reply accounts for no files: its frontmatter has no files:")
	}
	if strings.TrimSpace(fm.Body) == "" {
		return "", "", errors.New("the design in the reply is a spine with no document under it")
	}
	return doc, rules, nil
}

// unfence removes a code fence a session wrapped a whole part in.
func unfence(s string) string {
	t := strings.TrimSpace(s)
	if !strings.HasPrefix(t, "```") {
		return s
	}
	nl := strings.Index(t, "\n")
	if nl < 0 {
		return s
	}
	fence := strings.TrimRight(t[:nl], "abcdefghijklmnopqrstuvwxyz \t")
	if !strings.HasSuffix(t, "\n"+fence) {
		return s
	}
	return t[nl+1 : len(t)-len(fence)]
}

// project puts the project's rules into CLAUDE.md between the two marks,
// replacing what was between them and nothing else; a file without the marks
// has them added at its end.
func project(old, rules string) string {
	block := ProjectBegin + "\n" + strings.TrimSpace(rules) + "\n" + ProjectEnd
	i, j := strings.Index(old, ProjectBegin), strings.Index(old, ProjectEnd)
	if i >= 0 && j > i {
		return old[:i] + block + old[j+len(ProjectEnd):]
	}
	if strings.TrimSpace(old) == "" {
		return "## This project\n\n" + block + "\n"
	}
	return strings.TrimRight(old, "\n") + "\n\n## This project\n\n" + block + "\n"
}
