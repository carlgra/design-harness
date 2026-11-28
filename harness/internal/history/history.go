// Package history reads a repository's past changes from git: what each one
// touched, and what a file held before and after. A change is a commit on the
// first-parent line of HEAD set against its first parent, so that where
// branches are merged a change is the merge, the whole pull request, and
// where they are squashed it is the squashed commit.
package history

import (
	"bufio"
	"fmt"
	"io"
	"os/exec"
	"path"
	"sort"
	"strings"
)

// emptyTree is the tree with nothing in it, which the first commit of a
// repository is set against.
const emptyTree = "4b825dc642cb6eb9a060e54bf8d69288fbee4904"

// Change is one path a commit touched. Old is empty for a file added, New
// for one deleted; they differ for one renamed.
type Change struct {
	Old, New string
}

// Commit is one past change.
type Commit struct {
	Hash    string
	Parent  string // the first parent, or the empty tree
	Date    string // YYYY-MM-DD, the day it was committed
	Subject string
	Changes []Change
}

// Short is the hash as a person reads it.
func (c Commit) Short() string {
	if len(c.Hash) > 8 {
		return c.Hash[:8]
	}
	return c.Hash
}

// Paths are every path the commit touched, old and new.
func (c Commit) Paths() []string {
	seen := map[string]bool{}
	var out []string
	for _, ch := range c.Changes {
		for _, p := range []string{ch.Old, ch.New} {
			if p != "" && !seen[p] {
				seen[p] = true
				out = append(out, p)
			}
		}
	}
	return out
}

func git(root string, args ...string) (string, error) {
	out, err := exec.Command("git", append([]string{"-C", root}, args...)...).Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok && len(ee.Stderr) > 0 {
			return "", fmt.Errorf("git %s: %s", args[0], strings.TrimSpace(string(ee.Stderr)))
		}
		return "", fmt.Errorf("git %s: %w", args[0], err)
	}
	return string(out), nil
}

// Commits are the last n changes on the first-parent line of HEAD, newest
// first, each with the paths it touched.
func Commits(root string, n int) ([]Commit, error) {
	out, err := git(root, "log", "--first-parent", fmt.Sprintf("-n%d", n), "--format=%H%x09%P%x09%cs%x09%s", "HEAD")
	if err != nil {
		return nil, err
	}
	var commits []Commit
	for _, line := range strings.Split(strings.TrimRight(out, "\n"), "\n") {
		f := strings.SplitN(line, "\t", 4)
		if len(f) < 4 {
			continue
		}
		c := Commit{Hash: f[0], Parent: emptyTree, Date: f[2], Subject: f[3]}
		if parents := strings.Fields(f[1]); len(parents) > 0 {
			c.Parent = parents[0]
		}
		diff, err := git(root, "diff", "--name-status", "-z", "-M", c.Parent, c.Hash)
		if err != nil {
			// A shallow clone ends at a commit whose parent is not there.
			break
		}
		c.Changes = parseStatus(diff)
		commits = append(commits, c)
	}
	return commits, nil
}

// parseStatus reads git diff --name-status -z: a status, then one path, or
// two for a rename or a copy.
func parseStatus(s string) []Change {
	var out []Change
	f := strings.Split(s, "\x00")
	for i := 0; i+1 < len(f); {
		status := f[i]
		if status == "" {
			i++
			continue
		}
		switch status[0] {
		case 'R':
			if i+2 < len(f) {
				out = append(out, Change{Old: f[i+1], New: f[i+2]})
			}
			i += 3
		case 'C':
			if i+2 < len(f) {
				out = append(out, Change{New: f[i+2]})
			}
			i += 3
		case 'A':
			out = append(out, Change{New: f[i+1]})
			i += 2
		case 'D':
			out = append(out, Change{Old: f[i+1]})
			i += 2
		default:
			out = append(out, Change{Old: f[i+1], New: f[i+1]})
			i += 2
		}
	}
	return out
}

// Blobs reads files as they were at a commit, through one git process.
type Blobs struct {
	cmd *exec.Cmd
	in  io.WriteCloser
	out *bufio.Reader
}

// Open starts the reader.
func Open(root string) (*Blobs, error) {
	cmd := exec.Command("git", "-C", root, "cat-file", "--batch")
	in, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	out, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	return &Blobs{cmd: cmd, in: in, out: bufio.NewReaderSize(out, 1<<16)}, nil
}

// Read is the content of path at rev; nil where there is no such file.
func (b *Blobs) Read(rev, path string) []byte {
	if path == "" || strings.ContainsAny(path, "\n") {
		return nil
	}
	if _, err := fmt.Fprintf(b.in, "%s:%s\n", rev, path); err != nil {
		return nil
	}
	head, err := b.out.ReadString('\n')
	if err != nil {
		return nil
	}
	f := strings.Fields(head)
	if len(f) < 3 || f[len(f)-2] != "blob" {
		if len(f) >= 3 {
			// Not a file: a directory or a submodule. Its content is skipped.
			var size int
			fmt.Sscanf(f[len(f)-1], "%d", &size)
			io.CopyN(io.Discard, b.out, int64(size)+1)
		}
		return nil
	}
	var size int
	fmt.Sscanf(f[len(f)-1], "%d", &size)
	buf := make([]byte, size)
	if _, err := io.ReadFull(b.out, buf); err != nil {
		return nil
	}
	b.out.ReadByte()
	return buf
}

// Close stops the reader.
func (b *Blobs) Close() {
	b.in.Close()
	b.cmd.Wait()
}

// Pair is two things that changed in the same commits.
type Pair struct {
	A, B string
	// Together is the number of commits that touched both; OfA and OfB the
	// number that touched each.
	Together, OfA, OfB int
}

// Share is how much of the rarer one's changes also touched the other.
func (p Pair) Share() float64 {
	least := p.OfA
	if p.OfB < least {
		least = p.OfB
	}
	if least == 0 {
		return 0
	}
	return float64(p.Together) / float64(least)
}

// Together counts, over the commits, how often each two groups changed in
// one commit. group names the group a path belongs to; a path it gives no
// name is left out. A commit that touches more than wide groups is a sweep,
// a rename or a reformat, and says nothing of what belongs together: it is
// left out. The second value is how many commits touched each group.
func Together(commits []Commit, group func(string) string, wide int) ([]Pair, map[string]int) {
	of := map[string]int{}
	both := map[[2]string]int{}
	for _, c := range commits {
		seen := map[string]bool{}
		for _, p := range c.Paths() {
			if g := group(p); g != "" {
				seen[g] = true
			}
		}
		if len(seen) > wide {
			continue
		}
		var gs []string
		for g := range seen {
			gs = append(gs, g)
			of[g]++
		}
		sort.Strings(gs)
		for i := range gs {
			for j := i + 1; j < len(gs); j++ {
				both[[2]string{gs[i], gs[j]}]++
			}
		}
	}
	var pairs []Pair
	for k, n := range both {
		pairs = append(pairs, Pair{A: k[0], B: k[1], Together: n, OfA: of[k[0]], OfB: of[k[1]]})
	}
	sort.Slice(pairs, func(i, j int) bool {
		if pairs[i].Together != pairs[j].Together {
			return pairs[i].Together > pairs[j].Together
		}
		if pairs[i].A != pairs[j].A {
			return pairs[i].A < pairs[j].A
		}
		return pairs[i].B < pairs[j].B
	})
	return pairs, of
}

// Dir is the directory a path is grouped under where nothing else groups
// it: its first two elements, or one for a file near the root.
func Dir(p string) string {
	parts := strings.Split(p, "/")
	switch {
	case len(parts) == 1:
		return "(root)"
	case len(parts) == 2:
		return parts[0]
	}
	return path.Join(parts[0], parts[1])
}
