// Command backtest tries the design against the repository's own history
// before anybody is held to it, and changes nothing.
//
// It answers two questions of the last changes on the default branch. Of the
// areas the design divides the code into: how many changes stayed inside one,
// and which areas keep changing together, since an area is a part a change
// can stay inside. Of each rule that has a check: which of those changes
// would it have refused, and for what, since a rule adopted blind is found
// too strict or too loose afterwards.
//
//	go -C harness run ./cmd/backtest                 # the last 200 changes
//	go -C harness run ./cmd/backtest -n 1000 -since 2026-01-01
//	go -C harness run ./cmd/backtest -rule INV-007   # one rule, with every change it would have refused
//	go -C harness run ./cmd/backtest -forbid 'from "\.\./billing' -files 'src/**'   # a rule not yet written down
//	go -C harness run ./cmd/backtest -run            # command rules too, in a checkout of each change
//
// The rules and the areas are the ones in the design as it stands in the
// working tree, committed or not, so a rule or a boundary can be tried
// before it is proposed.
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"harness/internal/design"
	"harness/internal/history"
	"harness/internal/rules"
)

func main() {
	rootFlag := flag.String("root", "", "repository root (default: walk up from the working directory to the first directory holding design/DESIGN.md)")
	nFlag := flag.Int("n", 200, "how many changes to go back")
	sinceFlag := flag.String("since", "", "go back no further than this day, YYYY-MM-DD")
	ruleFlag := flag.String("rule", "", "only this rule, by id (an area's as area/INV-001), with every change it would have refused")
	runFlag := flag.Bool("run", false, "also try rules that are commands, by checking out each change in a worktree of its own: slow")
	forbidFlag := flag.String("forbid", "", "try a pattern no line may match, as a rule not yet in the design")
	requireFlag := flag.String("require", "", "try a pattern every file must hold, as a rule not yet in the design")
	filesFlag := flag.String("files", "", "the globs the tried pattern reads, comma-separated (default: every file)")
	exceptFlag := flag.String("except", "", "globs the tried pattern leaves out, comma-separated")
	reportFlag := flag.String("report", "", "also write the result as Markdown to this file")
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
	if !design.IsRepo(root) {
		fatal(fmt.Errorf("%s is not a git repository: there is no history to try the design against", root))
	}

	trial := *forbidFlag != "" || *requireFlag != ""
	if trial {
		// A rule being tried stands alone: nothing else is reported.
		repo.Design.Rules = []design.Rule{{
			ID: "INV-000", Forbid: *forbidFlag, Require: *requireFlag,
			Files: commas(*filesFlag), Except: commas(*exceptFlag),
		}}
		for i := range repo.Spines {
			repo.Spines[i].Rules = nil
		}
	}
	list, wrong := rules.Compile(repo)
	if len(wrong) > 0 {
		fatal(fmt.Errorf("%s", strings.Join(wrong, "\n")))
	}
	if *ruleFlag != "" {
		var one []*rules.Rule
		for _, r := range list {
			if r.Name() == *ruleFlag {
				one = append(one, r)
			}
		}
		if len(one) == 0 {
			fatal(fmt.Errorf("%s: no rule of that name has a check (rules: in the design or a spine)", *ruleFlag))
		}
		list = one
	}

	commits, err := history.CommitsSince(root, *nFlag, *sinceFlag)
	if err != nil {
		fatal(err)
	}
	if len(commits) == 0 {
		fatal(fmt.Errorf("no commits to read"))
	}

	asked := *nFlag
	if *sinceFlag != "" {
		asked = 0
	}
	rep := Report{Repo: repo.Name, Commits: commits, Asked: asked, Trial: trial, Detail: *ruleFlag != "" || trial}
	if !trial && *ruleFlag == "" {
		rep.Areas = scoreAreas(repo, commits)
	}
	if len(list) > 0 {
		rep.Rules, err = tryRules(repo, list, commits, *runFlag)
		if err != nil {
			fatal(err)
		}
	}
	fmt.Print(rep.Text())
	if *reportFlag != "" {
		if err := os.WriteFile(*reportFlag, []byte(rep.Markdown()), 0o644); err != nil {
			fatal(err)
		}
	}
}

func commas(s string) []string {
	var out []string
	for _, f := range strings.Split(s, ",") {
		if f = strings.TrimSpace(f); f != "" {
			out = append(out, f)
		}
	}
	return out
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "backtest:", err)
	os.Exit(2)
}
