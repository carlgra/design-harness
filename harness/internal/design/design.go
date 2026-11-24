// Package design reads a repository's one design document, design/DESIGN.md,
// and walks the repository for the files it must account for.
//
// The document is current state: what the repository is for, how it works,
// what must always hold, every decision in force with its reason and what was
// rejected, and which files realise it. Its frontmatter, the spine, is the
// small part a machine reads: the globs the code lives under, the tests that
// must exist, the repository's own gate, and what a design note must say.
package design

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// Design is the frontmatter of design/DESIGN.md: the spine.
type Design struct {
	// Files are the globs the repository's code lives under. Every source
	// file must match one; a file that matches none is unaccounted for.
	Files []string `yaml:"files"`
	// Tests are names that must appear in a test file under Files: each walks
	// a path a fault would travel.
	Tests []string `yaml:"tests"`
	// Validate is the repository's own gate, a command run from the root
	// after the checks pass. Optional.
	Validate string `yaml:"validate"`
	// Commits is the convention a commit's subject follows here; the only
	// one is "conventional". Empty binds nothing.
	Commits string `yaml:"commits"`
	// Notes says what a design note under design/changes/ must carry.
	Notes Notes `yaml:"notes"`
	// Repos are the repositories nested under this one, which is then a
	// root: the map. sync reads it, and the checker holds the disk to it.
	Repos []RepoEntry `yaml:"repos"`
}

// RepoEntry is one repository nested under a root.
type RepoEntry struct {
	// Path is where it is checked out, relative to the root.
	Path string `yaml:"path"`
	// Kind says what it is, in the root's own words: service, contracts,
	// platform. Nothing in the harness gives a kind meaning yet.
	Kind string `yaml:"kind"`
	// Remote is where it is cloned from. Empty means it is local only.
	Remote string `yaml:"remote"`
	// Status is "planned" for one that is decided and does not exist yet.
	Status string `yaml:"status"`
}

// Notes are the rules a design note is held to.
type Notes struct {
	// From is the first date, YYYY-MM-DD, a note is held to the rules. A note
	// dated before it is never reread: the rules bind what is written after
	// they exist. Empty binds every note.
	From string `yaml:"from"`
	// Sections are the headings every bound note carries, each with
	// something under it. Empty means DefaultSections.
	Sections []string `yaml:"sections"`
}

// DefaultSections are the four parts of a design note.
var DefaultSections = []string{"Problem", "Plan", "Alternatives rejected", "Files"}

// Repo is everything a harness command knows about one repository.
type Repo struct {
	Root   string // absolute path
	Name   string // the repository's name: its origin remote's, else the directory's
	Design Design
	Body   string // the document after its frontmatter

	// NestedRepos are directories, relative to Root, that hold their own .git.
	// Commands record their names and never descend into them.
	NestedRepos []string
	// Files are every regular file, relative to Root, outside .git, design/
	// and nested repositories. These are what Design.Files must account for.
	Files []string
	// Dirs are every directory walked, relative to Root.
	Dirs []string
}

// Parent is the root a nested repository sits under: the nearest directory
// above it that holds a design and a harness.
type Parent struct {
	Root   string // absolute path
	Name   string
	Design Design
	Body   string
	Doc    string // the whole document, as a session is handed it
	// Path is the nested repository's path relative to the root, and Up the
	// way back: "billing" and "..", or "services/billing" and "../..".
	Path string
	Up   string
}

// FindParent finds the root a repository is nested under, or nil when it
// stands alone. A root is a directory above it that holds both a design and
// the harness's Go module; the nearest one is the root.
func FindParent(root string) (*Parent, error) {
	for dir := filepath.Dir(root); ; dir = filepath.Dir(dir) {
		if IsRoot(dir) {
			if _, err := os.Stat(filepath.Join(dir, "harness", "go.mod")); err == nil {
				b, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(DocPath)))
				if err != nil {
					return nil, err
				}
				fm, err := ParseFrontmatter(string(b))
				if err != nil {
					return nil, fmt.Errorf("%s: %w", filepath.Join(dir, DocPath), err)
				}
				p := &Parent{Root: dir, Name: NameOf(dir), Body: fm.Body, Doc: string(b), Path: Rel(dir, root)}
				if err := yaml.Unmarshal([]byte(fm.YAML), &p.Design); err != nil {
					return nil, fmt.Errorf("%s: frontmatter: %w", filepath.Join(dir, DocPath), err)
				}
				p.Up = strings.TrimSuffix(strings.Repeat("../", strings.Count(p.Path, "/")+1), "/")
				return p, nil
			}
		}
		if filepath.Dir(dir) == dir {
			return nil, nil
		}
	}
}

// DocPath is where the design lives, relative to a root.
const DocPath = "design/DESIGN.md"

// Load reads the design and walks the repository.
func Load(root string) (*Repo, error) {
	r := &Repo{Root: root, Name: NameOf(root)}
	fm, err := ReadFrontmatterFile(filepath.Join(root, filepath.FromSlash(DocPath)))
	if err != nil {
		return nil, fmt.Errorf("%s: %w", DocPath, err)
	}
	if err := yaml.Unmarshal([]byte(fm.YAML), &r.Design); err != nil {
		return nil, fmt.Errorf("%s: frontmatter: %w", DocPath, err)
	}
	r.Body = fm.Body
	if err := r.walk(); err != nil {
		return nil, err
	}
	return r, nil
}

// Walk walks a repository that may have no design yet: the files a design
// would have to account for. The reverse-engineering survey starts here.
func Walk(root string) (*Repo, error) {
	r := &Repo{Root: root, Name: NameOf(root)}
	if err := r.walk(); err != nil {
		return nil, err
	}
	return r, nil
}

// walk records every file and directory a command must account for. It
// skips .git, other worktrees of this repository, and stops at the boundary
// of any nested repository. In a git repository the files are git's view:
// tracked, plus untracked and not ignored.
func (r *Repo) walk() error {
	err := filepath.WalkDir(r.Root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == r.Root {
			return nil
		}
		relp := Rel(r.Root, path)
		if d.IsDir() {
			if d.Name() == ".git" || IsWorktree(path) {
				return filepath.SkipDir
			}
			r.Dirs = append(r.Dirs, relp)
			if IsRepo(path) {
				r.NestedRepos = append(r.NestedRepos, relp)
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasPrefix(relp, "design/") {
			return nil
		}
		r.Files = append(r.Files, relp)
		return nil
	})
	if err != nil || !IsRepo(r.Root) {
		return err
	}
	return r.walkGit()
}

func (r *Repo) walkGit() error {
	cmd := exec.Command("git", "-C", r.Root, "ls-files", "-z", "--cached", "--others", "--exclude-standard")
	out, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("git ls-files: %w", err)
	}
	nested := func(p string) bool {
		for _, n := range r.NestedRepos {
			if p == n || strings.HasPrefix(p, n+"/") {
				return true
			}
		}
		return false
	}
	r.Files = r.Files[:0]
	dirs := map[string]bool{}
	for _, p := range strings.Split(string(out), "\x00") {
		if p == "" || strings.HasSuffix(p, "/") || strings.HasPrefix(p, "design/") || nested(p) {
			continue
		}
		if _, err := os.Lstat(filepath.Join(r.Root, filepath.FromSlash(p))); err != nil {
			continue
		}
		r.Files = append(r.Files, p)
		for d := filepath.Dir(p); d != "." && d != "/"; d = filepath.Dir(d) {
			dirs[d] = true
		}
	}
	r.Dirs = r.Dirs[:0]
	for d := range dirs {
		r.Dirs = append(r.Dirs, d)
	}
	r.Dirs = append(r.Dirs, r.NestedRepos...)
	sort.Strings(r.Dirs)
	sort.Strings(r.Files)
	return nil
}

// NameOf is a repository's name: the last element of its origin remote,
// without .git, so that a worktree checked out under another directory name
// is still the repository it is. A directory with no origin is named for
// itself.
func NameOf(root string) string {
	if !IsRepo(root) {
		return filepath.Base(root)
	}
	out, err := exec.Command("git", "-C", root, "remote", "get-url", "origin").Output()
	if url := strings.TrimSpace(string(out)); err == nil && url != "" {
		url = strings.TrimSuffix(strings.TrimSuffix(url, "/"), ".git")
		if i := strings.LastIndexAny(url, "/:"); i >= 0 {
			url = url[i+1:]
		}
		if url != "" {
			return url
		}
	}
	return filepath.Base(root)
}

// --- the repository root ---

// FindRoot resolves the repository root: an explicit one as given, otherwise
// the working directory or the nearest parent holding design/DESIGN.md. The
// working directory is the shell's $PWD when set, since `go -C harness` moves
// the process into the harness while $PWD still names the directory the
// command was typed in.
func FindRoot(explicit string) (string, error) {
	dir := WorkingDir()
	if explicit != "" {
		abs, err := Abs(explicit)
		if err != nil {
			return "", err
		}
		if !IsRoot(abs) {
			return "", fmt.Errorf("%s is not a repository root (no %s)", abs, DocPath)
		}
		return abs, nil
	}
	for {
		if IsRoot(dir) {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", errors.New("no repository root found: no parent directory holds " + DocPath)
		}
		dir = parent
	}
}

// FindProject resolves the root of a repository that may have no design yet:
// an explicit one as given, otherwise the git repository the command was
// typed in.
func FindProject(explicit string) (string, error) {
	if explicit != "" {
		abs, err := Abs(explicit)
		if err != nil {
			return "", err
		}
		if !IsDir(abs) {
			return "", fmt.Errorf("%s is not a directory", abs)
		}
		return abs, nil
	}
	out, err := exec.Command("git", "-C", WorkingDir(), "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return "", errors.New("not in a git repository: name the project with -root")
	}
	return strings.TrimSpace(string(out)), nil
}

// Abs is p as an absolute path, read against the directory the command was
// typed in.
func Abs(p string) (string, error) {
	if !filepath.IsAbs(p) {
		p = filepath.Join(WorkingDir(), p)
	}
	return filepath.Clean(p), nil
}

// IsRoot reports whether dir holds the design document.
func IsRoot(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, filepath.FromSlash(DocPath)))
	return err == nil
}

// FindHarness returns the one harness for the repository at root: root's own
// harness/ if it holds the Go module, else the nearest ancestor's, so that
// several repositories nested under one root can share the root's. A link is
// not a harness, whatever it resolves to.
func FindHarness(root string) (string, error) {
	for dir := root; ; dir = filepath.Dir(dir) {
		h := filepath.Join(dir, "harness")
		if fi, err := os.Lstat(h); err == nil && fi.Mode()&os.ModeSymlink == 0 {
			if _, err := os.Stat(filepath.Join(h, "go.mod")); err == nil {
				return h, nil
			}
		}
		if filepath.Dir(dir) == dir {
			return "", fmt.Errorf("no harness found in %s or above it", root)
		}
	}
}

// WorkingDir is the directory the command was run from: $PWD when the shell
// set it and it exists, otherwise the process's own.
func WorkingDir() string {
	if pwd := os.Getenv("PWD"); filepath.IsAbs(pwd) && IsDir(pwd) {
		return pwd
	}
	wd, err := os.Getwd()
	if err != nil {
		return "."
	}
	return wd
}

var gitLocalEnv = []string{
	"GIT_ALTERNATE_OBJECT_DIRECTORIES", "GIT_CONFIG", "GIT_CONFIG_PARAMETERS",
	"GIT_CONFIG_COUNT", "GIT_OBJECT_DIRECTORY", "GIT_DIR", "GIT_WORK_TREE",
	"GIT_IMPLICIT_WORK_TREE", "GIT_GRAFT_FILE", "GIT_INDEX_FILE",
	"GIT_NO_REPLACE_OBJECTS", "GIT_REPLACE_REF_BASE", "GIT_PREFIX",
	"GIT_SHALLOW_FILE", "GIT_COMMON_DIR",
}

// ForgetGitEnv drops the repository git handed this process, so that every
// git a command runs finds its repository from the directory it is given. A
// hook runs with GIT_DIR set, and from a linked worktree that names the outer
// repository for every nested one.
func ForgetGitEnv() {
	for _, k := range gitLocalEnv {
		os.Unsetenv(k)
	}
}

// IsRepo reports whether dir holds its own .git.
func IsRepo(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, ".git"))
	return err == nil
}

// IsWorktree reports whether dir is a linked git worktree: another checkout
// of a repository, never a repository of its own.
func IsWorktree(dir string) bool {
	b, err := os.ReadFile(filepath.Join(dir, ".git"))
	if err != nil {
		return false
	}
	line := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(string(b)), "gitdir:"))
	return strings.Contains(filepath.ToSlash(line), "/.git/worktrees/")
}

// IsDir reports whether p exists and is a directory.
func IsDir(p string) bool {
	st, err := os.Stat(p)
	return err == nil && st.IsDir()
}

// --- frontmatter ---

// Frontmatter is a markdown file split into its leading YAML and the rest.
type Frontmatter struct {
	YAML string
	Body string
}

// ReadFrontmatterFile reads and splits one file.
func ReadFrontmatterFile(path string) (Frontmatter, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return Frontmatter{}, err
	}
	return ParseFrontmatter(string(b))
}

// ParseFrontmatter splits a document into its leading YAML block and the body.
func ParseFrontmatter(s string) (Frontmatter, error) {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	if !strings.HasPrefix(s, "---\n") {
		return Frontmatter{}, errors.New("missing frontmatter: the file must start with ---")
	}
	rest := s[len("---\n"):]
	if i := strings.Index(rest, "\n---\n"); i >= 0 {
		return Frontmatter{YAML: rest[:i], Body: rest[i+len("\n---\n"):]}, nil
	}
	if strings.HasSuffix(rest, "\n---") {
		return Frontmatter{YAML: strings.TrimSuffix(rest, "\n---"), Body: ""}, nil
	}
	return Frontmatter{}, errors.New("unterminated frontmatter: no closing ---")
}

// Rel returns p relative to root with forward slashes.
func Rel(root, p string) string {
	r, err := filepath.Rel(root, p)
	if err != nil {
		return filepath.ToSlash(p)
	}
	return filepath.ToSlash(r)
}

// --- globs ---

// Exempt are root files that are part of how a repository works rather than
// its code, and so need no entry in the design.
var Exempt = map[string]bool{
	"README.md":      true,
	"CLAUDE.md":      true,
	"AGENTS.md":      true,
	".gitignore":     true,
	".gitattributes": true,
	"LICENSE":        true,
	"LICENSE.md":     true,
	"LICENSE.txt":    true,
}

// Claimed reports whether any of the globs accounts for f.
func Claimed(globs []string, f string) bool {
	for _, g := range globs {
		if MatchGlob(g, f) {
			return true
		}
	}
	return false
}

// MatchGlob supports a trailing /** for "everything under this directory" and
// otherwise path.Match semantics, in which * does not cross a /.
func MatchGlob(g, f string) bool {
	if strings.HasSuffix(g, "/**") {
		dir := strings.TrimSuffix(g, "/**")
		return f == dir || strings.HasPrefix(f, dir+"/")
	}
	ok, _ := filepath.Match(filepath.FromSlash(g), filepath.FromSlash(f))
	return ok
}

// IsTestFile recognises the test-file conventions of Go, Python, Rust's
// tests directory and the JavaScript runners.
func IsTestFile(f string) bool {
	b := filepath.Base(f)
	switch {
	case strings.HasSuffix(b, "_test.go"):
		return true
	case strings.HasSuffix(b, "_test.py"), strings.HasPrefix(b, "test_") && strings.HasSuffix(b, ".py"):
		return true
	case strings.Contains(b, ".test."), strings.Contains(b, ".spec."):
		return true
	case strings.HasSuffix(b, "_test.rs"), strings.HasSuffix(b, "Test.java"), strings.HasSuffix(b, "Test.kt"), strings.HasSuffix(b, "_spec.rb"), strings.HasSuffix(b, "_test.rb"):
		return true
	}
	return false
}
