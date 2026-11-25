// Command protection reads how the server protects a repository's default
// branch, in a repository whose design says `protection: required`, and
// refuses when a pull request, the `harness` status check or the refusal of
// a force push is missing; it passes everything in one that does not say so.
//
//	go -C harness run ./cmd/protection         # read it; safe-push does, before every push
//	go -C harness run ./cmd/protection -set    # add what is lacking, and change nothing else
//	go -C harness run ./cmd/protection -set -n # print what -set would send, and send nothing
//
// The hook and the push script refuse a push to main on a person's machine.
// What refuses a merge that never went near them is a setting on the server,
// made by hand, and this is what looks at it again. It reads through the gh
// command-line tool, as whoever is logged in to it, and on GitHub only.
package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"

	"harness/internal/design"
)

// Required is the one value `protection:` takes.
const Required = "required"

func main() {
	rootFlag := flag.String("root", "", "repository root (default: walk up from the working directory to the first directory holding design/DESIGN.md)")
	setFlag := flag.Bool("set", false, "add what the branch's protection lacks, and change nothing else")
	dryFlag := flag.Bool("n", false, "with -set: print what would be sent to the server, and send nothing")
	flag.Parse()
	design.ForgetGitEnv()

	root, err := design.FindRoot(*rootFlag)
	if err != nil {
		fatal(err)
	}
	asked, err := Asked(root)
	if err != nil {
		fatal(err)
	}
	if !asked && !*setFlag {
		return
	}
	repo, branch, err := origin(root)
	if err != nil {
		fatal(err)
	}
	if _, err := exec.LookPath("gh"); err != nil {
		fatal(errors.New("the gh command-line tool is not installed, and the server's protection is read through it"))
	}

	if *setFlag {
		set(repo, branch, *dryFlag)
		if !asked {
			fmt.Printf("protection: this repository's design does not ask for it to be read. Add  protection: %s  to the frontmatter of %s and every push will.\n", Required, design.DocPath)
		}
		return
	}

	s, err := Read(gh, repo, branch)
	if err != nil {
		fatal(err)
	}
	lacking := s.Lacking()
	switch {
	case len(lacking) == 0:
		fmt.Printf("protection: %s on the server requires a pull request and the `%s` check, and refuses a force push\n", branch, Check)
	case !s.Readable:
		// Whoever cannot read a branch's protection cannot change it either,
		// and is not refused for a setting that is not theirs.
		fmt.Printf("protection: the server shows how %s is protected to an administrator only, and no ruleset shown to you makes up for it (%s). Not refused: an administrator's push reads it.\n", branch, strings.Join(lacking, "; "))
	default:
		fmt.Fprintf(os.Stderr, "harness: %s is not protected on the server as this repository's design asks (%s, protection: %s):\n", branch, design.DocPath, Required)
		for _, l := range lacking {
			fmt.Fprintln(os.Stderr, "  - "+l)
		}
		fmt.Fprintf(os.Stderr, "  To add what is lacking and change nothing else:\n      go -C %s run ./cmd/protection -set\n  Or by hand: https://github.com/%s/settings/branches\n", harnessPath(root), repo)
		os.Exit(1)
	}
}

func set(repo, branch string, dry bool) {
	if dry {
		s, err := Read(gh, repo, branch)
		if err != nil {
			fatal(err)
		}
		if len(s.Lacking()) == 0 {
			fmt.Printf("protection: %s lacks nothing; nothing would be sent\n", branch)
			return
		}
		body, err := s.Body()
		if err != nil {
			fatal(err)
		}
		fmt.Printf("protection: would send to repos/%s/branches/%s/protection:\n%s\n", repo, branch, body)
		return
	}
	s, err := Set(gh, repo, branch)
	if err != nil {
		fatal(err)
	}
	if lacking := s.Lacking(); len(lacking) > 0 {
		fatal(fmt.Errorf("the server took the change and %s still lacks: %s", branch, strings.Join(lacking, "; ")))
	}
	fmt.Printf("protection: %s on the server requires a pull request and the `%s` check, and refuses a force push\n", branch, Check)
}

// Asked reports whether the design at root asks for the server's protection
// to be read.
func Asked(root string) (bool, error) {
	fm, err := design.ReadFrontmatterFile(filepath.Join(root, filepath.FromSlash(design.DocPath)))
	if err != nil {
		return false, fmt.Errorf("%s: %w", design.DocPath, err)
	}
	var d design.Design
	if err := yaml.Unmarshal([]byte(fm.YAML), &d); err != nil {
		return false, fmt.Errorf("%s: frontmatter: %w", design.DocPath, err)
	}
	switch d.Protection {
	case "":
		return false, nil
	case Required:
		return true, nil
	}
	return false, fmt.Errorf("%s: protection: %q is not a value the harness knows; the only one is %q", design.DocPath, d.Protection, Required)
}

var githubRepo = regexp.MustCompile(`github\.com[:/]([^/]+/[^/]+?)(\.git)?/?$`)

// origin is the repository on GitHub as owner/name, and its default branch.
func origin(root string) (repo, branch string, err error) {
	out, e := exec.Command("git", "-C", root, "remote", "get-url", "origin").Output()
	url := strings.TrimSpace(string(out))
	if e != nil || url == "" {
		return "", "", errors.New("this repository has no origin whose protection could be read")
	}
	m := githubRepo.FindStringSubmatch(url)
	if m == nil {
		return "", "", fmt.Errorf("the server's protection is read on GitHub only, and origin is %s. Take  protection: %s  out of the design's frontmatter", url, Required)
	}
	branch = "main"
	if out, e := exec.Command("git", "-C", root, "symbolic-ref", "--quiet", "--short", "refs/remotes/origin/HEAD").Output(); e == nil {
		if b := strings.TrimPrefix(strings.TrimSpace(string(out)), "origin/"); b != "" {
			branch = b
		}
	}
	return m[1], branch, nil
}

var httpStatus = regexp.MustCompile(`\(HTTP (\d{3})\)`)

// gh is the API through the gh command-line tool. A reply the server gave
// with an error status is a reply, not a failure to ask.
func gh(method, path string, body []byte) ([]byte, int, error) {
	args := []string{"api", "--method", method, path}
	if body != nil {
		args = append(args, "--input", "-")
	}
	cmd := exec.Command("gh", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if body != nil {
		cmd.Stdin = bytes.NewReader(body)
	}
	err := cmd.Run()
	if err == nil {
		return stdout.Bytes(), 200, nil
	}
	if m := httpStatus.FindStringSubmatch(stderr.String()); m != nil {
		status, _ := strconv.Atoi(m[1])
		return stdout.Bytes(), status, nil
	}
	if msg := strings.TrimSpace(stderr.String()); msg != "" {
		return nil, 0, errors.New(msg)
	}
	return nil, 0, err
}

// harnessPath is the harness as it is reached from root: its own, or the
// one above it.
func harnessPath(root string) string {
	if h, err := design.FindHarness(root); err == nil {
		if rel, err := filepath.Rel(root, h); err == nil {
			return filepath.ToSlash(rel)
		}
	}
	return "harness"
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "harness: "+err.Error())
	os.Exit(2)
}
