package main

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// server is a GitHub that holds one branch's protection and one list of
// ruleset rules, and records what is written to it.
type server struct {
	protection string // the GET reply; "" is an unprotected branch
	status     int    // of the protection GET, when not 200 or the 404 of ""
	rules      string
	down       bool
	put        []byte
}

func (s *server) api(method, path string, body []byte) ([]byte, int, error) {
	if s.down {
		return nil, 0, errors.New("could not resolve host")
	}
	switch {
	case method == "GET" && path == "repos/acme/work":
		return []byte(`{}`), 200, nil
	case method == "GET" && strings.HasSuffix(path, "/branches/main/protection"):
		if s.status != 0 {
			return []byte(`{"message":"Not Found"}`), s.status, nil
		}
		if s.protection == "" {
			return []byte(`{"message":"Branch not protected"}`), 404, nil
		}
		return []byte(s.protection), 200, nil
	case method == "GET" && strings.HasSuffix(path, "/rules/branches/main"):
		if s.rules == "" {
			return []byte(`[]`), 200, nil
		}
		return []byte(s.rules), 200, nil
	case method == "PUT" && strings.HasSuffix(path, "/branches/main/protection"):
		s.put = body
		// What a server does with a body: holds it, in the shape it gives back.
		var b struct {
			StatusChecks struct {
				Strict bool    `json:"strict"`
				Checks []check `json:"checks"`
			} `json:"required_status_checks"`
			Reviews     json.RawMessage `json:"required_pull_request_reviews"`
			ForcePushes bool            `json:"allow_force_pushes"`
		}
		if err := json.Unmarshal(body, &b); err != nil {
			return []byte(`{"message":"Invalid request"}`), 422, nil
		}
		held := map[string]any{
			"required_status_checks":        map[string]any{"strict": b.StatusChecks.Strict, "checks": b.StatusChecks.Checks},
			"required_pull_request_reviews": b.Reviews,
			"allow_force_pushes":            map[string]bool{"enabled": b.ForcePushes},
		}
		out, _ := json.Marshal(held)
		s.protection = string(out)
		return out, 200, nil
	}
	return []byte(`{"message":"Not Found"}`), 404, nil
}

const whole = `{
  "required_status_checks": {"strict": true, "contexts": ["harness"], "checks": [{"context": "harness", "app_id": 15368}]},
  "required_pull_request_reviews": {"required_approving_review_count": 0},
  "enforce_admins": {"enabled": true},
  "allow_force_pushes": {"enabled": false}
}`

// A branch is held to a pull request, the harness check and no force push,
// by its own protection or by a ruleset, and each one missing is named.
func TestAProtectionThatLacksSomethingIsNamedForIt(t *testing.T) {
	for _, c := range []struct {
		name    string
		srv     server
		lacking []string
	}{
		{"all of it", server{protection: whole}, nil},
		{"no protection at all", server{}, []string{"a pull request is not required", "the `harness` status check is not required", "a force push is not refused"}},
		{"a pull request and no check", server{protection: `{"required_pull_request_reviews": {"required_approving_review_count": 1}, "allow_force_pushes": {"enabled": false}}`},
			[]string{"the `harness` status check is not required"}},
		{"another check, not the harness's", server{protection: `{"required_status_checks": {"contexts": ["lint"]}, "required_pull_request_reviews": {}, "allow_force_pushes": {"enabled": false}}`},
			[]string{"the `harness` status check is not required"}},
		{"force pushes allowed", server{protection: `{"required_status_checks": {"contexts": ["harness"]}, "required_pull_request_reviews": {}, "allow_force_pushes": {"enabled": true}}`},
			[]string{"a force push is not refused"}},
		{"a ruleset holds all of it", server{rules: `[{"type":"pull_request"},{"type":"non_fast_forward"},{"type":"required_status_checks","parameters":{"required_status_checks":[{"context":"harness"}]}}]`}, nil},
		{"a ruleset makes up what the protection lacks", server{protection: `{"required_pull_request_reviews": {}, "allow_force_pushes": {"enabled": false}}`,
			rules: `[{"type":"required_status_checks","parameters":{"required_status_checks":[{"context":"harness"}]}}]`}, nil},
	} {
		t.Run(c.name, func(t *testing.T) {
			s, err := Read(c.srv.api, "acme/work", "main")
			if err != nil {
				t.Fatal(err)
			}
			if !s.Readable {
				t.Error("read as not shown")
			}
			if got := s.Lacking(); !reflect.DeepEqual(got, c.lacking) {
				t.Errorf("lacking %q, want %q", got, c.lacking)
			}
		})
	}
}

// A protection the server will not show is not taken for one that is absent,
// and a server that cannot be asked is an error, never a pass.
func TestAProtectionThatCannotBeReadIsNotTakenForNone(t *testing.T) {
	for _, status := range []int{403, 404} {
		srv := server{status: status}
		s, err := Read(srv.api, "acme/work", "main")
		if err != nil {
			t.Fatal(err)
		}
		if s.Readable {
			t.Errorf("HTTP %d was read as a branch with no protection", status)
		}
		if _, err := s.Body(); err == nil {
			t.Errorf("HTTP %d: a protection that was not read would be overwritten", status)
		}
	}
	// Not shown, and a ruleset anyone may read holds all of it.
	srv := server{status: 404, rules: `[{"type":"pull_request"},{"type":"non_fast_forward"},{"type":"required_status_checks","parameters":{"required_status_checks":[{"context":"harness"}]}}]`}
	if s, err := Read(srv.api, "acme/work", "main"); err != nil || len(s.Lacking()) != 0 {
		t.Errorf("a ruleset was not read: %v, %v", s.Lacking(), err)
	}
	down := server{down: true}
	if _, err := Read(down.api, "acme/work", "main"); err == nil {
		t.Error("a server that could not be asked was read as an answer")
	}
	broke := server{status: 500}
	if _, err := Read(broke.api, "acme/work", "main"); err == nil {
		t.Error("a server error was read as an answer")
	}
}

// Setting adds what is lacking and hands back everything else as it was.
func TestSettingAddsWhatIsLackingAndChangesNothingElse(t *testing.T) {
	t.Run("an unprotected branch gets the least that will do", func(t *testing.T) {
		srv := server{}
		s, err := Set(srv.api, "acme/work", "main")
		if err != nil || len(s.Lacking()) != 0 {
			t.Fatalf("%v, %v", s.Lacking(), err)
		}
		var sent map[string]any
		if err := json.Unmarshal(srv.put, &sent); err != nil {
			t.Fatal(err)
		}
		if got := sent["required_pull_request_reviews"].(map[string]any)["required_approving_review_count"]; got != float64(0) {
			t.Errorf("approvals %v: one person could then not merge their own pull request", got)
		}
		if v, ok := sent["restrictions"]; !ok || v != nil {
			t.Errorf("restrictions %v, want null", v)
		}
		if sent["enforce_admins"] != false || sent["allow_force_pushes"] != false {
			t.Errorf("sent %s", srv.put)
		}
	})

	t.Run("what is there is kept", func(t *testing.T) {
		srv := server{protection: `{
		  "required_status_checks": {"strict": true, "contexts": ["lint"], "checks": [{"context": "lint", "app_id": 7}]},
		  "required_pull_request_reviews": {"required_approving_review_count": 2, "dismiss_stale_reviews": true, "require_code_owner_reviews": true},
		  "enforce_admins": {"enabled": true},
		  "restrictions": {"users": [{"login": "ada"}], "teams": [{"slug": "core"}], "apps": []},
		  "required_linear_history": {"enabled": true},
		  "required_conversation_resolution": {"enabled": true},
		  "allow_force_pushes": {"enabled": true}
		}`}
		if _, err := Set(srv.api, "acme/work", "main"); err != nil {
			t.Fatal(err)
		}
		var sent map[string]any
		json.Unmarshal(srv.put, &sent)
		seven := float64(7)
		want := map[string]any{
			"required_status_checks": map[string]any{"strict": true, "checks": []any{
				map[string]any{"context": "lint", "app_id": seven}, map[string]any{"context": "harness"}}},
			"enforce_admins": true,
			"required_pull_request_reviews": map[string]any{"dismiss_stale_reviews": true, "require_code_owner_reviews": true,
				"required_approving_review_count": float64(2), "require_last_push_approval": false},
			"restrictions":                     map[string]any{"users": []any{"ada"}, "teams": []any{"core"}, "apps": []any{}},
			"required_linear_history":          true,
			"allow_force_pushes":               false,
			"allow_deletions":                  false,
			"block_creations":                  false,
			"required_conversation_resolution": true,
			"lock_branch":                      false,
			"allow_fork_syncing":               false,
		}
		if !reflect.DeepEqual(sent, want) {
			t.Errorf("sent\n%s", srv.put)
		}
	})

	t.Run("a branch that lacks nothing is not written to", func(t *testing.T) {
		srv := server{protection: whole}
		if _, err := Set(srv.api, "acme/work", "main"); err != nil || srv.put != nil {
			t.Errorf("wrote %s, %v", srv.put, err)
		}
	})

	t.Run("what cannot be handed back whole is not overwritten", func(t *testing.T) {
		srv := server{protection: `{"required_pull_request_reviews": {"dismissal_restrictions": {"users": [{"login": "ada"}]}}, "allow_force_pushes": {"enabled": false}}`}
		if _, err := Set(srv.api, "acme/work", "main"); err == nil || srv.put != nil {
			t.Errorf("wrote %s, %v", srv.put, err)
		}
	})
}

// The design asks for the server's protection to be read, or it is not read.
func TestProtectionIsReadOnlyWhereTheDesignAsks(t *testing.T) {
	for frontmatter, want := range map[string]bool{"": false, "protection: required\n": true} {
		root := t.TempDir()
		os.MkdirAll(filepath.Join(root, "design"), 0o755)
		os.WriteFile(filepath.Join(root, "design", "DESIGN.md"), []byte("---\nfiles:\n  - app/**\n"+frontmatter+"---\n\n# x\n"), 0o644)
		got, err := Asked(root)
		if err != nil || got != want {
			t.Errorf("%q: asked %v, %v", frontmatter, got, err)
		}
	}
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, "design"), 0o755)
	os.WriteFile(filepath.Join(root, "design", "DESIGN.md"), []byte("---\nprotection: yes please\n---\n\n# x\n"), 0o644)
	if _, err := Asked(root); err == nil {
		t.Error("a value the harness does not know was taken for one it does")
	}
}
