package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"harness/internal/design"
	"harness/internal/history"
	"harness/internal/rules"
)

// RuleTrial is one rule's check tried against the history.
type RuleTrial struct {
	Name  string
	Kind  string
	Check string // the pattern or the command
	From  string
	// Reads is the number of files the check reads in the tree as it
	// stands; a check that reads none holds nothing.
	Reads int
	// Now is the number of breaches in the tree as it stands, and Known the
	// length of the rule's list.
	Now   int
	Known int
	// Added and Mended are the breaches the changes brought in and took
	// out. Refused are the changes that brought one in, newest first: each
	// would have been refused had the rule been in force.
	Added   int
	Mended  int
	Refused []Refusal
	// Skipped is why the check was not tried, when it was not.
	Skipped string
	// Failed is the number of changes at which a command could not be run.
	Failed int
}

// Start is the number of breaches there were before the first change read.
func (t RuleTrial) Start() int {
	if n := t.Now - t.Added + t.Mended; n > 0 {
		return n
	}
	return 0
}

// Refusal is a change a rule would have refused, and the breaches it added.
type Refusal struct {
	Commit history.Commit
	Added  []rules.Finding
	Mended int
}

// tryRules tries each check against each change. A pattern is tried by
// reading every file the change touched as it was before and after: a breach
// the change added is one found after and not before. A command is tried
// only when asked, since it needs the whole tree of each change checked out.
func tryRules(repo *design.Repo, list []*rules.Rule, commits []history.Commit, run bool) ([]RuleTrial, error) {
	trials := make([]RuleTrial, len(list))
	now, failed := rules.Evaluate(repo, patternsAnd(list, run))
	for i, r := range list {
		t := &trials[i]
		t.Name, t.Kind, t.From = r.Name(), r.Kind(), r.From
		t.Check = r.Forbid + r.Require + r.Run + strings.Join(r.Absent, " ")
		known, _, _ := rules.ReadKnown(repo.Root, r)
		t.Known = len(known)
		if r.Kind() == "run" {
			if !run {
				t.Skipped = "a command, tried only with -run"
			} else if err := failed[r]; err != nil {
				t.Skipped = err.Error()
			}
			t.Now = len(now[r])
			continue
		}
		t.Now = len(now[r])
		for _, f := range repo.Files {
			if r.Reads(f) {
				t.Reads++
			}
		}
	}

	blobs, err := history.Open(repo.Root)
	if err != nil {
		return nil, err
	}
	defer blobs.Close()
	for _, c := range commits {
		added := make([][]rules.Finding, len(list))
		mended := make([]int, len(list))
		for _, ch := range c.Changes {
			// The documents under design/ are not code, now or then.
			if strings.HasPrefix(ch.Old, "design/") {
				ch.Old = ""
			}
			if strings.HasPrefix(ch.New, "design/") {
				ch.New = ""
			}
			var before, after []byte
			var read bool
			for i, r := range list {
				if r.Kind() == "run" {
					continue
				}
				readsOld, readsNew := ch.Old != "" && r.Reads(ch.Old), ch.New != "" && r.Reads(ch.New)
				if !readsOld && !readsNew {
					continue
				}
				if !read {
					before, after, read = blobs.Read(c.Parent, ch.Old), blobs.Read(c.Hash, ch.New), true
				}
				var was, is []rules.Finding
				if readsOld && before != nil {
					was = r.Scan(ch.Old, before)
				}
				if readsNew && after != nil {
					is = r.Scan(ch.New, after)
				}
				a, m := brought(was, is)
				added[i] = append(added[i], a...)
				mended[i] += m
			}
		}
		for i := range list {
			trials[i].Added += len(added[i])
			trials[i].Mended += mended[i]
			if len(added[i]) > 0 {
				trials[i].Refused = append(trials[i].Refused, Refusal{Commit: c, Added: added[i], Mended: mended[i]})
			}
		}
	}
	if run {
		if err := tryCommands(repo, list, commits, trials); err != nil {
			return nil, err
		}
	}
	return trials, nil
}

func patternsAnd(list []*rules.Rule, run bool) []*rules.Rule {
	var out []*rules.Rule
	for _, r := range list {
		if run || r.Kind() != "run" {
			out = append(out, r)
		}
	}
	return out
}

// brought sets what a file held after a change against what it held before,
// by what was found and not by where: a line that moved is the same breach,
// and so is every breach of a file that was renamed.
func brought(was, is []rules.Finding) (added []rules.Finding, mended int) {
	left := map[string]int{}
	for _, f := range was {
		left[f.Text]++
	}
	for _, f := range is {
		if left[f.Text] > 0 {
			left[f.Text]--
			continue
		}
		added = append(added, f)
	}
	for _, n := range left {
		mended += n
	}
	return added, mended
}

// tryCommands runs each command rule at each change, oldest first, in a
// worktree of its own that is removed afterwards. What a command prints at
// one change, set against what it printed at the one before, is what that
// change added.
func tryCommands(repo *design.Repo, list []*rules.Rule, commits []history.Commit, trials []RuleTrial) error {
	any := false
	for _, r := range list {
		any = any || r.Kind() == "run"
	}
	if !any {
		return nil
	}
	tmp, err := os.MkdirTemp("", "backtest-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)
	tree := filepath.Join(tmp, "tree")
	git := func(dir string, args ...string) error {
		out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).CombinedOutput()
		if err != nil {
			return fmt.Errorf("git %s: %s", args[0], strings.TrimSpace(string(out)))
		}
		return nil
	}
	oldest := commits[len(commits)-1]
	first := oldest.Parent
	if exec.Command("git", "-C", repo.Root, "cat-file", "-e", first+"^{commit}").Run() != nil {
		// The oldest change read is the repository's first commit: there is
		// no tree before it, and it is the starting point.
		first = oldest.Hash
	}
	if err := git(repo.Root, "worktree", "add", "--quiet", "--detach", tree, first); err != nil {
		return err
	}
	defer git(repo.Root, "worktree", "remove", "--force", tree)

	prev := map[int][]string{}
	okPrev := map[int]bool{}
	runAll := func() map[int][]rules.Finding {
		out := map[int][]rules.Finding{}
		for i, r := range list {
			if r.Kind() != "run" {
				continue
			}
			f, err := r.RunAt(tree)
			if err != nil {
				out[i] = nil
				okPrev[i] = false
				trials[i].Failed++
				continue
			}
			out[i] = f
		}
		return out
	}
	for i, f := range runAll() {
		prev[i], okPrev[i] = rules.Keys(f), trials[i].Failed == 0
	}
	for i := range trials {
		trials[i].Failed = 0
	}
	for n := len(commits) - 1; n >= 0; n-- {
		c := commits[n]
		if c.Hash == first {
			continue
		}
		if err := git(tree, "checkout", "--quiet", "--detach", c.Hash); err != nil {
			return err
		}
		failedBefore := map[int]int{}
		for i := range trials {
			failedBefore[i] = trials[i].Failed
		}
		for i, found := range runAll() {
			if trials[i].Failed > failedBefore[i] {
				continue
			}
			keys := rules.Keys(found)
			if okPrev[i] {
				fresh, _ := rules.Split(found, prev[i])
				gone := rules.Minus(prev[i], keys)
				trials[i].Added += len(fresh)
				trials[i].Mended += len(gone)
				if len(fresh) > 0 {
					trials[i].Refused = append([]Refusal{{Commit: c, Added: fresh, Mended: len(gone)}}, trials[i].Refused...)
				}
			}
			prev[i], okPrev[i] = keys, true
		}
	}
	return nil
}
