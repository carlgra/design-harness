// Command branch says whether a name is one a change branch may have in this
// repository: feat/<what> or fix/<what>, or, where the design's frontmatter
// sets `branches:`, a name the whole of which matches that regular
// expression.
//
//	go -C harness run ./cmd/branch feat/a-fourth-line   # exits 0 when it is, 1 when it is not
//	go -C harness run ./cmd/branch -form                # print what a change branch is called here
//
// safe-push runs it on the branch it is asked to push, and on each name it
// would move stray work to. The scripts are shell and match nothing
// themselves.
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
	formFlag := flag.Bool("form", false, "print what a change branch is called here, and check nothing")
	flag.Parse()
	design.ForgetGitEnv()

	root, err := design.FindRoot(*rootFlag)
	if err != nil {
		fatal(err)
	}
	rule, err := convention.Branches(root)
	if err != nil {
		fatal(err)
	}
	if *formFlag {
		fmt.Println(rule.Form())
		return
	}
	if flag.NArg() != 1 {
		fatal(fmt.Errorf("usage: branch NAME | branch -form"))
	}
	if name := flag.Arg(0); !rule.Matches(name) {
		fmt.Fprintf(os.Stderr, "harness: %s is not a change branch's name here: a change branch is named %s\n", name, rule.Form())
		os.Exit(1)
	}
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "harness: "+err.Error())
	os.Exit(2)
}
