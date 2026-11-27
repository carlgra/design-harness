// Command check holds a repository to its design, design/DESIGN.md and the
// spine of each area it defines: that every source file is under a path the
// design accounts for, and under one area at most, that every test the design names exists, that every decision
// and invariant id is defined once and every one cited is defined, and that
// every design note carries its parts. Then it runs the repository's own
// gate, if the design declares one.
//
//	go -C harness run ./cmd/check
//	go -C harness run ./cmd/check -validate=false   # the checks only
//	go -C harness run ./cmd/check -for src/cart     # which documents to read before changing these paths
//
// The pre-push hook and safe-push run it before every push; CI runs it as the
// status check main requires. Everything it does not check is read by a
// person.
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
	rootFlag := flag.String("root", "", "repository root (default: walk up from the working directory to the first directory holding design/DESIGN.md)")
	validateFlag := flag.Bool("validate", true, "run the design's validate: command after the checks pass")
	forFlag := flag.Bool("for", false, "print the design documents to read before changing the paths given, and check nothing")
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

	if *forFlag {
		for _, doc := range documentsFor(repo, flag.Args()) {
			fmt.Println(doc)
		}
		return
	}

	res := runChecks(repo)
	res.report(os.Stdout, repo)
	if res.failed() {
		os.Exit(1)
	}

	if *validateFlag && repo.Design.Validate != "" {
		fmt.Printf("harness: validate: %s\n", repo.Design.Validate)
		cmd := exec.Command("sh", "-c", repo.Design.Validate)
		cmd.Dir = repo.Root
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			fmt.Println("harness: validate: FAILED")
			os.Exit(1)
		}
		fmt.Println("harness: validate: OK")
	}
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "harness:", err)
	os.Exit(2)
}

// documentsFor are the documents to read before changing the paths given:
// the design, always, then the spine of every area that holds one of them. A
// path is a file or a directory, as typed from the repository's root or from
// the directory the command was typed in.
func documentsFor(repo *design.Repo, paths []string) []string {
	docs := []string{design.DocPath}
	want := map[string]bool{}
	for _, p := range paths {
		if abs, err := design.Abs(p); err == nil {
			if rel := design.Rel(repo.Root, abs); !strings.HasPrefix(rel, "..") {
				p = rel
			}
		}
		p = strings.Trim(filepath.ToSlash(p), "/")
		for _, n := range repo.AreasOf(p) {
			want[n] = true
		}
		for _, f := range repo.Files {
			if strings.HasPrefix(f, p+"/") {
				for _, n := range repo.AreasOf(f) {
					want[n] = true
				}
			}
		}
	}
	for _, a := range repo.Design.Spines {
		if want[a.Name] {
			docs = append(docs, design.SpinePath(a.Name))
		}
	}
	return docs
}
