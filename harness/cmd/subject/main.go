// Command subject says whether a commit's subject is a Conventional Commit,
// in a repository whose design says `commits: conventional`; it passes
// everything in one that does not.
//
//	go -C harness run ./cmd/subject "feat(billing): a thing"   # a commit's subject
//	go -C harness run ./cmd/subject -file .git/COMMIT_EDITMSG  # as the commit-msg hook is handed it
//	go -C harness run ./cmd/subject -title "feat: a thing"     # a pull request's title
//
// The commit-msg hook runs it on every commit and safe-push runs it on the
// title of the pull request, which is what a squash merge puts on main.
package main

import (
	"flag"
	"fmt"
	"os"

	"harness/internal/convention"
	"harness/internal/design"
)

func main() {
	rootFlag := flag.String("root", "", "repository root (default: walk up from the working directory to the first directory holding design/DESIGN.md)")
	fileFlag := flag.String("file", "", "read the subject from a commit message file, as the commit-msg hook is handed one")
	titleFlag := flag.Bool("title", false, "the subject is a pull request's title: none of git's own subjects is excused")
	flag.Parse()
	design.ForgetGitEnv()

	root, err := design.FindRoot(*rootFlag)
	if err != nil {
		fatal(err)
	}
	binds, err := convention.Binds(root)
	if err != nil {
		fatal(err)
	}
	if !binds {
		return
	}

	var subject string
	switch {
	case *fileFlag != "":
		b, err := os.ReadFile(*fileFlag)
		if err != nil {
			fatal(err)
		}
		subject = convention.SubjectOf(string(b))
	case flag.NArg() == 1:
		subject = flag.Arg(0)
	default:
		fatal(fmt.Errorf("usage: subject [-title] SUBJECT | subject -file COMMIT_MSG_FILE"))
	}

	if why := convention.Refusal(subject, *titleFlag); why != "" {
		fmt.Fprintln(os.Stderr, "harness: "+why)
		fmt.Fprintln(os.Stderr, "  This repository's design asks for them (design/DESIGN.md, commits: conventional).")
		os.Exit(1)
	}
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "harness: "+err.Error())
	os.Exit(2)
}
