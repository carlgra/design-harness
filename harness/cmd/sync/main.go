// Command sync makes the disk match a root's map, the repos: list in its
// design. For each entry it clones a repository that is missing, fetches one
// that is on disk, and skips anything planned. It never merges and never
// touches a working tree. A fresh machine is rebuilt with:
//
//	git clone <the root> project && cd project
//	go -C harness run ./cmd/sync
package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"harness/internal/design"
)

func main() {
	rootFlag := flag.String("root", "", "repository root (default: walk up from the working directory)")
	dryRun := flag.Bool("dry-run", false, "say what would be done and do nothing")
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
	if len(repo.Design.Repos) == 0 {
		fmt.Println("sync: the design lists no nested repositories; nothing to do")
		return
	}

	failed := 0
	for _, e := range repo.Design.Repos {
		p := strings.Trim(e.Path, "/")
		dir := filepath.Join(root, filepath.FromSlash(p))
		action, detail := plan(e, design.IsDir(dir))
		say(action, p, detail)
		if *dryRun {
			continue
		}
		switch action {
		case "missing":
			failed++
		case "clone":
			err = run(root, "git", "clone", "--", e.Remote, dir)
		case "fetch":
			err = run(dir, "git", "fetch", "--prune")
		}
		if err != nil {
			say("error", p, err.Error())
			failed++
			err = nil
		}
	}

	if failed > 0 {
		fmt.Fprintf(os.Stderr, "sync: %d of the map's repositories could not be brought in line\n", failed)
		os.Exit(1)
	}
	if *dryRun {
		fmt.Println("sync: dry run; nothing changed")
	}
}

// plan says what one entry needs: nothing is merged and no working tree is
// touched, whatever the answer.
func plan(e design.RepoEntry, onDisk bool) (action, detail string) {
	switch {
	case e.Status == "planned":
		return "skip", "planned"
	case e.Remote == "" && onDisk:
		return "local-only", "no remote recorded; left as is"
	case e.Remote == "":
		return "missing", "no remote recorded; cannot clone"
	case !onDisk:
		return "clone", e.Remote
	}
	return "fetch", e.Remote
}

func say(action, path, detail string) {
	fmt.Printf("%-10s %-24s %s\n", action, path, detail)
}

func run(dir, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "sync:", err)
	os.Exit(2)
}
