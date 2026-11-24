// Command check holds a repository to its one design document,
// design/DESIGN.md: that every source file is under a path the design
// accounts for, that every test the design names exists, that every decision
// and invariant id is defined once and every one cited is defined, and that
// every design note carries its parts. Then it runs the repository's own
// gate, if the design declares one.
//
//	go -C harness run ./cmd/check
//	go -C harness run ./cmd/check -validate=false   # the checks only
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

	"harness/internal/design"
)

func main() {
	rootFlag := flag.String("root", "", "repository root (default: walk up from the working directory to the first directory holding design/DESIGN.md)")
	validateFlag := flag.Bool("validate", true, "run the design's validate: command after the checks pass")
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
