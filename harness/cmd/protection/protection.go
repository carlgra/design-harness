package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
)

// API is one call to the server's REST API: the method, the path, and a body
// or nil. status is the HTTP status of a reply the server gave, 0 when none
// arrived, and then err says why. The tests replace it.
type API func(method, path string, body []byte) (out []byte, status int, err error)

// Check is the status check the method's CI workflow reports under: the name
// of its one job.
const Check = "harness"

// State is what the server holds a branch to, read from two places: the
// branch's own protection, which only an administrator may read, and the
// rules of every ruleset that applies to it, which anyone may.
type State struct {
	// Readable is false when the server would not show the branch's
	// protection to whoever asked.
	Readable bool
	// PullRequest: nothing reaches the branch except through one.
	PullRequest bool
	// Checks are the status checks a pull request must pass.
	Checks []string
	// NoForcePush: a force push to the branch is refused.
	NoForcePush bool
	// classic is the branch's protection as the server gave it, nil when it
	// has none or it could not be read.
	classic *classic
}

type toggle struct {
	Enabled bool `json:"enabled"`
}

type check struct {
	Context string `json:"context"`
	AppID   *int   `json:"app_id,omitempty"`
}

type actors struct {
	Users []struct {
		Login string `json:"login"`
	} `json:"users"`
	Teams []struct {
		Slug string `json:"slug"`
	} `json:"teams"`
	Apps []struct {
		Slug string `json:"slug"`
	} `json:"apps"`
}

func (a *actors) empty() bool {
	return a == nil || len(a.Users)+len(a.Teams)+len(a.Apps) == 0
}

// classic is a branch's protection as GET .../protection returns it: the
// parts this command reads, and the parts it must hand back unchanged when
// it writes.
type classic struct {
	StatusChecks *struct {
		Strict   bool     `json:"strict"`
		Contexts []string `json:"contexts"`
		Checks   []check  `json:"checks"`
	} `json:"required_status_checks"`
	EnforceAdmins toggle `json:"enforce_admins"`
	Reviews       *struct {
		DismissStale    bool    `json:"dismiss_stale_reviews"`
		CodeOwners      bool    `json:"require_code_owner_reviews"`
		Approvals       int     `json:"required_approving_review_count"`
		LastPush        bool    `json:"require_last_push_approval"`
		Dismissal       *actors `json:"dismissal_restrictions"`
		BypassAllowance *actors `json:"bypass_pull_request_allowances"`
	} `json:"required_pull_request_reviews"`
	Restrictions     *actors `json:"restrictions"`
	LinearHistory    toggle  `json:"required_linear_history"`
	ForcePushes      toggle  `json:"allow_force_pushes"`
	Deletions        toggle  `json:"allow_deletions"`
	BlockCreations   toggle  `json:"block_creations"`
	Conversations    toggle  `json:"required_conversation_resolution"`
	LockBranch       toggle  `json:"lock_branch"`
	AllowForkSyncing toggle  `json:"allow_fork_syncing"`
}

// Read asks the server what it holds branch to. An error is a server that
// could not be asked, or a repository it does not know: never a branch that
// is merely unprotected, which is a State with nothing in it.
func Read(api API, repo, branch string) (State, error) {
	var s State
	if _, status, err := api("GET", "repos/"+repo, nil); err != nil {
		return s, fmt.Errorf("%s cannot be reached through gh: %w", repo, err)
	} else if status != 200 {
		return s, fmt.Errorf("the server does not show %s to you (HTTP %d). Run  gh auth login", repo, status)
	}

	out, status, err := api("GET", "repos/"+repo+"/branches/"+branch+"/protection", nil)
	switch {
	case err != nil:
		return s, fmt.Errorf("the protection of %s cannot be read: %w", branch, err)
	case status == 200:
		var c classic
		if err := json.Unmarshal(out, &c); err != nil {
			return s, fmt.Errorf("the protection of %s, as the server gave it, cannot be read: %w", branch, err)
		}
		s.Readable, s.classic = true, &c
		s.PullRequest = c.Reviews != nil
		s.NoForcePush = !c.ForcePushes.Enabled
		if c.StatusChecks != nil {
			s.Checks = append(s.Checks, c.StatusChecks.Contexts...)
			for _, k := range c.StatusChecks.Checks {
				s.Checks = append(s.Checks, k.Context)
			}
		}
	case status == 404 && strings.Contains(message(out), "not protected"):
		// An administrator asked, and there is none.
		s.Readable = true
	case status == 403 || status == 404:
		// Not shown to whoever asked: it takes an administrator.
	default:
		return s, fmt.Errorf("the protection of %s cannot be read: HTTP %d: %s", branch, status, message(out))
	}

	// Rulesets hold a branch to the same things by another road, and their
	// rules are shown to anyone who can read the repository. A server that
	// has none of them, or will not say, adds nothing.
	if out, status, err := api("GET", "repos/"+repo+"/rules/branches/"+branch, nil); err == nil && status == 200 {
		var rules []struct {
			Type       string `json:"type"`
			Parameters struct {
				Checks []check `json:"required_status_checks"`
			} `json:"parameters"`
		}
		if json.Unmarshal(out, &rules) == nil {
			for _, r := range rules {
				switch r.Type {
				case "pull_request":
					s.PullRequest = true
				case "non_fast_forward":
					s.NoForcePush = true
				case "required_status_checks":
					for _, k := range r.Parameters.Checks {
						s.Checks = append(s.Checks, k.Context)
					}
				}
			}
		}
	}
	s.Checks = uniq(s.Checks)
	return s, nil
}

// Lacking lists what the method needs of a protected branch and s does not
// have, in words a person can act on. Empty when nothing is.
func (s State) Lacking() []string {
	var out []string
	if !s.PullRequest {
		out = append(out, "a pull request is not required")
	}
	if !has(s.Checks, Check) {
		out = append(out, fmt.Sprintf("the `%s` status check is not required", Check))
	}
	if !s.NoForcePush {
		out = append(out, "a force push is not refused")
	}
	return out
}

// Body is what to send the server so that the branch lacks nothing: the
// protection it has, with what is missing added and nothing else changed, or
// the least that will do where it has none. It refuses a protection it could
// not read, and one holding something it cannot hand back whole.
func (s State) Body() ([]byte, error) {
	if !s.Readable {
		return nil, errors.New("the branch's protection is not shown to you, so it cannot be changed by you either: it takes an administrator of the repository")
	}
	type reviews struct {
		DismissStale bool `json:"dismiss_stale_reviews"`
		CodeOwners   bool `json:"require_code_owner_reviews"`
		Approvals    int  `json:"required_approving_review_count"`
		LastPush     bool `json:"require_last_push_approval"`
	}
	type statusChecks struct {
		Strict bool    `json:"strict"`
		Checks []check `json:"checks"`
	}
	type restrictions struct {
		Users []string `json:"users"`
		Teams []string `json:"teams"`
		Apps  []string `json:"apps"`
	}
	body := struct {
		StatusChecks     statusChecks  `json:"required_status_checks"`
		EnforceAdmins    bool          `json:"enforce_admins"`
		Reviews          reviews       `json:"required_pull_request_reviews"`
		Restrictions     *restrictions `json:"restrictions"`
		LinearHistory    bool          `json:"required_linear_history"`
		ForcePushes      bool          `json:"allow_force_pushes"`
		Deletions        bool          `json:"allow_deletions"`
		BlockCreations   bool          `json:"block_creations"`
		Conversations    bool          `json:"required_conversation_resolution"`
		LockBranch       bool          `json:"lock_branch"`
		AllowForkSyncing bool          `json:"allow_fork_syncing"`
	}{}

	if c := s.classic; c != nil {
		if c.Reviews != nil {
			if !c.Reviews.Dismissal.empty() || !c.Reviews.BypassAllowance.empty() {
				return nil, errors.New("the branch's protection names who may dismiss a review or bypass a pull request, which this command cannot hand back unchanged: add what is lacking by hand, in the repository's settings")
			}
			body.Reviews = reviews{c.Reviews.DismissStale, c.Reviews.CodeOwners, c.Reviews.Approvals, c.Reviews.LastPush}
		}
		if c.StatusChecks != nil {
			body.StatusChecks.Strict = c.StatusChecks.Strict
			body.StatusChecks.Checks = append(body.StatusChecks.Checks, c.StatusChecks.Checks...)
			for _, name := range c.StatusChecks.Contexts {
				if !hasCheck(body.StatusChecks.Checks, name) {
					body.StatusChecks.Checks = append(body.StatusChecks.Checks, check{Context: name})
				}
			}
		}
		if r := c.Restrictions; r != nil {
			body.Restrictions = &restrictions{Users: []string{}, Teams: []string{}, Apps: []string{}}
			for _, u := range r.Users {
				body.Restrictions.Users = append(body.Restrictions.Users, u.Login)
			}
			for _, t := range r.Teams {
				body.Restrictions.Teams = append(body.Restrictions.Teams, t.Slug)
			}
			for _, a := range r.Apps {
				body.Restrictions.Apps = append(body.Restrictions.Apps, a.Slug)
			}
		}
		body.EnforceAdmins = c.EnforceAdmins.Enabled
		body.LinearHistory = c.LinearHistory.Enabled
		body.Deletions = c.Deletions.Enabled
		body.BlockCreations = c.BlockCreations.Enabled
		body.Conversations = c.Conversations.Enabled
		body.LockBranch = c.LockBranch.Enabled
		body.AllowForkSyncing = c.AllowForkSyncing.Enabled
	}
	if !hasCheck(body.StatusChecks.Checks, Check) {
		body.StatusChecks.Checks = append(body.StatusChecks.Checks, check{Context: Check})
	}
	body.ForcePushes = false
	return json.MarshalIndent(body, "", "  ")
}

// Set writes Body to the server and reads the branch again, so that what it
// reports is what the server now holds and not what was sent.
func Set(api API, repo, branch string) (State, error) {
	s, err := Read(api, repo, branch)
	if err != nil {
		return s, err
	}
	if len(s.Lacking()) == 0 {
		return s, nil
	}
	body, err := s.Body()
	if err != nil {
		return s, err
	}
	out, status, err := api("PUT", "repos/"+repo+"/branches/"+branch+"/protection", body)
	if err != nil {
		return s, fmt.Errorf("the protection of %s was not changed: %w", branch, err)
	}
	if status != 200 {
		return s, fmt.Errorf("the protection of %s was not changed: HTTP %d: %s", branch, status, message(out))
	}
	return Read(api, repo, branch)
}

// message is the server's own word for a refusal.
func message(out []byte) string {
	var m struct {
		Message string `json:"message"`
	}
	if json.Unmarshal(out, &m) == nil && m.Message != "" {
		return m.Message
	}
	return strings.TrimSpace(string(out))
}

func has(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

func hasCheck(list []check, name string) bool {
	for _, x := range list {
		if x.Context == name {
			return true
		}
	}
	return false
}

func uniq(list []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, x := range list {
		if !seen[x] {
			seen[x] = true
			out = append(out, x)
		}
	}
	sort.Strings(out)
	return out
}
