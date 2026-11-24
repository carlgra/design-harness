package main

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode"

	"gopkg.in/yaml.v3"

	"harness/internal/design"
)

// ContractsKind is the kind a root's map gives the repository that holds the
// definitions the others talk through.
const ContractsKind = "contracts"

// mergedRefs are where a repository's merged state is read, the first that
// resolves: what its remote's default branch held when last fetched, and not
// whatever branch happens to be checked out.
var mergedRefs = []string{"origin/HEAD", "origin/main", "origin/master"}

// Env is what the checker knows beyond the repository it checks: its place
// on the root's map, and where the contracts repository is.
type Env struct {
	// Role is "contracts" in the contracts repository, "server" in any other
	// repository under a root whose map lists one, and "" everywhere else,
	// where these rules bind nothing.
	Role string
	// Self is the repository's own path on the map: the name a servers entry
	// calls it by.
	Self string
	// ContractsDir is the contracts repository's checkout.
	ContractsDir string
	// Repos is the root's map.
	Repos []design.RepoEntry
}

// LocateEnv reads a repository's place from the root above it. A repository
// under no root, or under one whose map lists no contracts repository, has no
// role.
func LocateEnv(parent *design.Parent) *Env {
	env := &Env{}
	if parent == nil {
		return env
	}
	env.Self, env.Repos = parent.Path, parent.Design.Repos
	for _, e := range parent.Design.Repos {
		if !strings.EqualFold(e.Kind, ContractsKind) || e.Status == "planned" {
			continue
		}
		p := strings.Trim(e.Path, "/")
		if p == parent.Path {
			env.Role = "contracts"
		} else {
			env.Role = "server"
			env.ContractsDir = filepath.Join(parent.Root, filepath.FromSlash(p))
		}
		return env
	}
	return env
}

// --- reading a tree at a ref ---

// Tree is a set of files, by path relative to a repository, at one revision.
type Tree struct {
	At    string // the ref, or "the working tree"
	Files map[string]string
}

// readTree reads the files under dir whose paths keep accepts: at the merged
// state when dir is a git repository in which one of mergedRefs resolves,
// else at HEAD, else from the working tree.
func readTree(dir string, keep func(string) bool) (*Tree, error) {
	if design.IsRepo(dir) {
		for _, r := range append(append([]string{}, mergedRefs...), "HEAD") {
			if exec.Command("git", "-C", dir, "rev-parse", "--verify", "--quiet", r+"^{commit}").Run() != nil {
				continue
			}
			return readGitTree(dir, r, keep)
		}
	}
	t := &Tree{At: "the working tree", Files: map[string]string{}}
	err := filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == ".git" || d.Name() == "node_modules" {
				return filepath.SkipDir
			}
			return nil
		}
		rel := design.Rel(dir, p)
		if !keep(rel) {
			return nil
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		t.Files[rel] = string(b)
		return nil
	})
	return t, err
}

// mergedRef is the first of mergedRefs that resolves in dir, "" when none.
func mergedRef(dir string) string {
	for _, r := range mergedRefs {
		if exec.Command("git", "-C", dir, "rev-parse", "--verify", "--quiet", r+"^{commit}").Run() == nil {
			return r
		}
	}
	return ""
}

// hasOrigin reports whether dir is a git repository with a remote named
// origin.
func hasOrigin(dir string) bool {
	return design.IsRepo(dir) && exec.Command("git", "-C", dir, "remote", "get-url", "origin").Run() == nil
}

func readGitTree(dir, ref string, keep func(string) bool) (*Tree, error) {
	out, err := exec.Command("git", "-C", dir, "ls-tree", "-r", "-z", "--name-only", ref).Output()
	if err != nil {
		return nil, fmt.Errorf("git ls-tree %s in %s: %w", ref, dir, err)
	}
	t := &Tree{At: ref, Files: map[string]string{}}
	for _, p := range strings.Split(string(out), "\x00") {
		if p == "" || !keep(p) {
			continue
		}
		b, err := exec.Command("git", "-C", dir, "show", ref+":"+p).Output()
		if err != nil {
			return nil, fmt.Errorf("git show %s:%s in %s: %w", ref, p, dir, err)
		}
		t.Files[p] = string(b)
	}
	return t, nil
}

func isNote(p string) bool {
	return path.Dir(p) == ChangesDir && strings.HasSuffix(p, ".md")
}

// isProto is a definition a person wrote: generated and vendored copies are
// not the contract. Generated code is under gen/ at the top of the
// repository; a vendored copy is under a directory named vendor or
// node_modules, at any depth.
func isProto(p string) bool {
	if !strings.HasSuffix(p, ".proto") || strings.HasPrefix(p, "gen/") {
		return false
	}
	for _, seg := range strings.Split(path.Dir(p), "/") {
		if seg == "vendor" || seg == "node_modules" {
			return false
		}
	}
	return true
}

func (t *Tree) notes() []Note {
	var notes []Note
	for p, b := range t.Files {
		if isNote(p) {
			notes = append(notes, NewNote(p, b))
		}
	}
	sort.Slice(notes, func(i, j int) bool { return notes[i].Path < notes[j].Path })
	return notes
}

// --- the servers block ---

// Entry is one RPC or request field a contracts note lists: its fully
// qualified name, the repository that serves it and the ones that call it,
// each by its path on the root's map, and the note that listed it.
type Entry struct {
	Name    string   `yaml:"name"`
	Server  string   `yaml:"server"`
	Callers []string `yaml:"callers"`
	Note    string   `yaml:"-"`
}

var serversFence = regexp.MustCompile("^\\s*```+\\s*servers\\s*$")

// ParseServers reads every fenced block whose info string is `servers` in a
// note: a YAML list of entries. found is false when the note has none. A
// block inside an HTML comment is not a block: the note template shows one
// there as an example, and an example lists nothing. Nor is one inside a
// longer fence, which is a block being quoted.
func ParseServers(n Note) (entries []Entry, found bool, err error) {
	lines := strings.Split(htmlComment.ReplaceAllString(n.Body, ""), "\n")
	quoted := ""
	for i := 0; i < len(lines); i++ {
		t := strings.TrimSpace(lines[i])
		if quoted != "" {
			if t == quoted {
				quoted = ""
			}
			lines[i] = ""
			continue
		}
		if strings.HasPrefix(t, "````") {
			quoted = strings.TrimRight(t, "abcdefghijklmnopqrstuvwxyz \t")
			lines[i] = ""
		}
	}
	for i := 0; i < len(lines); i++ {
		if !serversFence.MatchString(lines[i]) {
			continue
		}
		found = true
		var block []string
		j := i + 1
		for ; j < len(lines) && !strings.HasPrefix(strings.TrimSpace(lines[j]), "```"); j++ {
			block = append(block, lines[j])
		}
		if j == len(lines) {
			return nil, true, fmt.Errorf("%s: the servers block is never closed", n.Path)
		}
		var list []Entry
		if err := yaml.Unmarshal([]byte(strings.Join(block, "\n")), &list); err != nil {
			return nil, true, fmt.Errorf("%s: the servers block is not a YAML list of name, server, callers: %v", n.Path, err)
		}
		for k := range list {
			list[k].Note = n.Path
		}
		entries = append(entries, list...)
		i = j
	}
	return entries, found, nil
}

func onMap(repos []design.RepoEntry, name string) bool {
	for _, r := range repos {
		if strings.Trim(r.Path, "/") == strings.Trim(name, "/") {
			return true
		}
	}
	return false
}

// --- the rules ---

// checkContracts holds a definition to having someone who serves it. In the
// contracts repository, every RPC and request field a branch adds is listed
// in a note with the repository that serves it and the ones that call it. In
// any other repository under the same root, every name the contracts
// repository has merged naming this one as server has a test in its design.
// At a root, and in a repository under no root or under one that keeps no
// contracts repository, nothing.
func checkContracts(repo *design.Repo, env *Env, res *Result) {
	switch env.Role {
	case "contracts":
		checkListed(repo, env, res)
	case "server":
		checkServed(repo, env, res)
	}
}

// checkListed is the contracts half. Its notes, from the date its spine
// holds notes to, carry a servers block; every entry names a server and
// callers on the root's map; and every RPC or request field the branch adds
// over what is merged is listed in some note.
func checkListed(repo *design.Repo, env *Env, res *Result) {
	notes, err := readNotes(repo.Root)
	if err != nil {
		res.fail("%s: %v", ChangesDir, err)
		return
	}
	listed := map[string]bool{}
	for _, n := range notes {
		entries, found, err := ParseServers(n)
		if err != nil {
			res.fail("%v", err)
			continue
		}
		if !found && n.Bound(repo.Design.Notes.From) {
			res.fail("%s: no servers block: list each new RPC and request field with the repository that serves it and the ones that call it, or `[]` when the change adds none", n.Path)
		}
		for _, e := range entries {
			validEntry(e, env.Repos, res)
			listed[e.Name] = true
		}
	}

	head := map[string]string{}
	for _, f := range repo.Files {
		if !isProto(f) {
			continue
		}
		b, err := os.ReadFile(filepath.Join(repo.Root, filepath.FromSlash(f)))
		if err != nil {
			res.fail("%s: %v", f, err)
			return
		}
		head[f] = string(b)
	}
	// What a branch adds is read against what is merged. Where the repository
	// has a remote and none of its merged refs is here, falling back to HEAD
	// would compare a branch with itself and pass everything: that is refused,
	// not skipped. A repository with no remote has merged nothing anywhere,
	// and its own HEAD is all there is to read against.
	if hasOrigin(repo.Root) && mergedRef(repo.Root) == "" {
		res.fail("cannot tell what this repository has merged: none of %s is here. Fetch, or say which branch is the default with  git remote set-head origin -a", strings.Join(mergedRefs, ", "))
		return
	}
	base, err := readTree(repo.Root, isProto)
	if err != nil {
		res.fail("cannot read the definitions already merged: %v", err)
		return
	}
	for _, name := range NewNames(ScanAll(base.Files), ScanAll(head)) {
		if !listed[name] {
			res.fail("%s is new over %s and no note in %s lists it: add it to the change note's servers block with its server and callers", name, base.At, ChangesDir)
		}
	}
}

func validEntry(e Entry, repos []design.RepoEntry, res *Result) {
	switch {
	case strings.TrimSpace(e.Name) == "":
		res.fail("%s: a servers entry with no name", e.Note)
	case e.Server == "":
		res.fail("%s: %s names no server", e.Note, e.Name)
	case !onMap(repos, e.Server):
		res.fail("%s: %s is served by %q, which the root's map does not list", e.Note, e.Name, e.Server)
	}
	if len(e.Callers) == 0 {
		res.fail("%s: %s names no caller: a definition nobody calls is not yet an interface", e.Note, e.Name)
	}
	for _, c := range e.Callers {
		if !onMap(repos, c) {
			res.fail("%s: %s is called by %q, which the root's map does not list", e.Note, e.Name, c)
		}
	}
}

// checkServed is the serving half. Every entry the contracts repository has
// merged naming this repository as server, whose name a merged definition
// declares, is under the design's serves: with a test, and every test serves:
// names exists in a test file the design accounts for. A serves: item
// contracts does not list is not refused, but its test must still exist.
func checkServed(repo *design.Repo, env *Env, res *Result) {
	if !design.IsDir(env.ContractsDir) {
		res.fail("the contracts repository is not checked out at %s, so what this repository must serve cannot be read: run sync at the root, or check it out there in CI", env.ContractsDir)
		return
	}
	tree, err := readTree(env.ContractsDir, isNote)
	if err != nil {
		res.fail("cannot read the contracts repository's notes: %v", err)
		return
	}
	protos, err := readTree(env.ContractsDir, isProto)
	if err != nil {
		res.fail("cannot read the contracts repository's definitions: %v", err)
		return
	}
	serves := map[string]design.Served{}
	for _, s := range repo.Design.Serves {
		serves[s.Name] = s
	}
	// A name binds once a merged definition declares it; until then the note
	// is a design, and a design binds nothing.
	merged := ScanAll(protos.Files)
	for _, n := range tree.notes() {
		entries, _, err := ParseServers(n)
		if err != nil {
			res.fail("the contracts repository at %s: %v", tree.At, err)
			continue
		}
		for _, e := range entries {
			if strings.Trim(e.Server, "/") != env.Self || !merged.Declares(e.Name) {
				continue
			}
			if s, ok := serves[e.Name]; !ok || strings.TrimSpace(s.Test) == "" {
				res.fail("%s: contracts (%s, at %s) says %s serves it, called by %s, and design/DESIGN.md serves: names no test for it",
					e.Name, e.Note, tree.At, env.Self, strings.Join(e.Callers, ", "))
			}
		}
	}
	tests := testFiles(repo)
	for _, s := range repo.Design.Serves {
		if strings.TrimSpace(s.Test) == "" {
			continue // refused above if contracts lists it
		}
		if !definedIn(repo.Root, tests, s.Test) {
			res.fail("design/DESIGN.md serves: %s with the test %s, which is in no test file it accounts for", s.Name, s.Test)
		}
	}
}

// --- the proto scan ---

// Surface is what a set of .proto files offers a caller: every RPC, by
// package, service and name; every field of every top-level request message,
// by package, message and field; and every field of every top-level message,
// request or not.
type Surface struct {
	RPCs     map[string]bool
	Requests map[string]map[string]bool // message -> fields
	Fields   map[string]bool            // package.Message.field
}

// Declares says whether name is an RPC or a field of a top-level message.
func (s Surface) Declares(name string) bool {
	return s.RPCs[name] || s.Fields[name]
}

// ScanAll scans every file and merges what they declare.
func ScanAll(files map[string]string) Surface {
	s := Surface{RPCs: map[string]bool{}, Requests: map[string]map[string]bool{}, Fields: map[string]bool{}}
	for _, src := range files {
		ScanProto(src, s)
	}
	return s
}

// NewNames is every RPC in head and not in base, and every field head adds to
// a request message base already had. A request message that is itself new
// is covered by the RPC that takes it.
func NewNames(base, head Surface) []string {
	var out []string
	for r := range head.RPCs {
		if !base.RPCs[r] {
			out = append(out, r)
		}
	}
	for m, fields := range head.Requests {
		old, existed := base.Requests[m]
		if !existed {
			continue
		}
		for f := range fields {
			if !old[f] {
				out = append(out, m+"."+f)
			}
		}
	}
	sort.Strings(out)
	return out
}

// ScanProto reads one .proto file into s. It is a scan, not a parser: enough
// of the grammar to find a package, services and their RPCs, top-level
// messages whose names end in Request, and their fields, oneofs included.
func ScanProto(src string, s Surface) {
	toks := tokenize(src)
	type frame struct{ kind, name string }
	var stack []frame
	pkg := ""
	qualify := func(n string) string {
		if pkg == "" {
			return n
		}
		return pkg + "." + n
	}
	var stmt []string
	brackets := 0
	top := func() frame {
		if len(stack) == 0 {
			return frame{kind: "file"}
		}
		return stack[len(stack)-1]
	}
	// message is the top-level message the scan is directly inside, if any:
	// the message itself or a oneof in it.
	message := func() string {
		n := len(stack)
		if n == 1 && stack[0].kind == "message" {
			return stack[0].name
		}
		if n == 2 && stack[0].kind == "message" && stack[1].kind == "oneof" {
			return stack[0].name
		}
		return ""
	}
	for i := 0; i < len(toks); i++ {
		t := toks[i]
		next := func(k int) string {
			if i+k < len(toks) {
				return toks[i+k]
			}
			return ""
		}
		switch t {
		case "[":
			brackets++
		case "]":
			brackets--
		case ";":
			stmt = stmt[:0]
			continue
		case "{":
			f := frame{kind: "block"}
			if len(stmt) >= 2 {
				switch stmt[0] {
				case "service", "message", "enum", "oneof":
					f = frame{kind: stmt[0], name: stmt[1]}
				}
			}
			stack = append(stack, f)
			stmt = stmt[:0]
			continue
		case "}":
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}
			stmt = stmt[:0]
			continue
		case "=":
			if brackets == 0 && len(stmt) > 0 {
				first := stmt[0]
				if m := message(); m != "" &&
					first != "option" && first != "reserved" && first != "extensions" {
					if name := stmt[len(stmt)-1]; isIdent(name) {
						s.Fields[qualify(m)+"."+name] = true
						if strings.HasSuffix(m, "Request") {
							s.Requests[qualify(m)][name] = true
						}
					}
				}
			}
		}
		if len(stmt) == 0 {
			switch t {
			case "package":
				if p := next(1); isIdent(p) {
					pkg = p
				}
			case "rpc":
				if f := top(); f.kind == "service" && isIdent(next(1)) {
					s.RPCs[qualify(f.name+"."+next(1))] = true
				}
			case "message":
				if len(stack) == 0 && strings.HasSuffix(next(1), "Request") {
					if s.Requests[qualify(next(1))] == nil {
						s.Requests[qualify(next(1))] = map[string]bool{}
					}
				}
			}
		}
		stmt = append(stmt, t)
	}
}

func isIdent(s string) bool {
	if s == "" {
		return false
	}
	for i, r := range s {
		if !(r == '_' || r == '.' || unicode.IsLetter(r) || (i > 0 && unicode.IsDigit(r))) {
			return false
		}
	}
	return true
}

// tokenize splits proto source into identifiers, numbers, strings and single
// punctuation, dropping comments.
func tokenize(src string) []string {
	var toks []string
	for i := 0; i < len(src); {
		c := src[i]
		switch {
		case c == '/' && i+1 < len(src) && src[i+1] == '/':
			for i < len(src) && src[i] != '\n' {
				i++
			}
		case c == '/' && i+1 < len(src) && src[i+1] == '*':
			end := strings.Index(src[i+2:], "*/")
			if end < 0 {
				return toks
			}
			i += end + 4
		case c == '"' || c == '\'':
			j := i + 1
			for j < len(src) && src[j] != c {
				if src[j] == '\\' {
					j++
				}
				j++
			}
			toks = append(toks, src[i:min(j+1, len(src))])
			i = j + 1
		case c == ' ' || c == '\t' || c == '\n' || c == '\r':
			i++
		case c == '_' || c == '.' || unicode.IsLetter(rune(c)) || unicode.IsDigit(rune(c)):
			j := i
			for j < len(src) && (src[j] == '_' || src[j] == '.' || unicode.IsLetter(rune(src[j])) || unicode.IsDigit(rune(src[j]))) {
				j++
			}
			toks = append(toks, src[i:j])
			i = j
		default:
			toks = append(toks, string(c))
			i++
		}
	}
	return toks
}
