package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"harness/internal/design"
)

var theMap = []design.RepoEntry{
	{Path: "contracts", Kind: "contracts"},
	{Path: "billing", Kind: "service"},
	{Path: "ledger", Kind: "service"},
}

const billingV1 = `syntax = "proto3";
// A comment with message FakeRequest { in it.
package billing.v1;

service BillingService {
  rpc ListInvoices(ListInvoicesRequest) returns (ListInvoicesResponse);
  rpc Settle(SettleRequest) returns (SettleResponse) {
    option idempotency_level = IDEMPOTENT;
  }
}

message ListInvoicesRequest {
  string customer = 1;
  int32 page_size = 2 [deprecated = true];
  map<string, string> labels = 3;
  oneof cursor {
    string after = 4;
  }
  message Nested { string inner = 1; }
  reserved 9;
}

message ListInvoicesResponse { repeated string items = 1; }
message SettleRequest { string key = 1; }
`

func writeFiles(t *testing.T, root string, files map[string]string) {
	t.Helper()
	for name, body := range files {
		p := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func git(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	for _, kv := range os.Environ() {
		if !strings.HasPrefix(kv, "GIT_") {
			cmd.Env = append(cmd.Env, kv)
		}
	}
	cmd.Env = append(cmd.Env, "GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

// A repository with files committed on main and recorded as origin/main, so
// that readTree reads what is merged.
func mergedRepo(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	git(t, dir, "init", "-q", "-b", "main")
	writeFiles(t, dir, files)
	git(t, dir, "add", "-A")
	git(t, dir, "commit", "-q", "--allow-empty", "-m", "merged")
	git(t, dir, "update-ref", "refs/remotes/origin/main", "HEAD")
	return dir
}

// The scan finds every RPC and every field of a top-level request message,
// oneofs included, and nothing from comments, options, reserved ranges,
// nested messages or responses.
func TestTheProtoScanFindsRPCsAndRequestFields(t *testing.T) {
	s := ScanAll(map[string]string{"billing/v1/billing.proto": billingV1})
	rpcs := map[string]bool{"billing.v1.BillingService.ListInvoices": true, "billing.v1.BillingService.Settle": true}
	if !reflect.DeepEqual(s.RPCs, rpcs) {
		t.Errorf("rpcs %v", s.RPCs)
	}
	want := map[string]map[string]bool{
		"billing.v1.ListInvoicesRequest": {"customer": true, "page_size": true, "labels": true, "after": true},
		"billing.v1.SettleRequest":       {"key": true},
	}
	if !reflect.DeepEqual(s.Requests, want) {
		t.Errorf("requests %v", s.Requests)
	}
	// A file with no package names its definitions without one.
	bare := ScanAll(map[string]string{"a.proto": "syntax = \"proto3\";\nservice S { rpc Do(DoRequest) returns (DoRequest); }\nmessage DoRequest { string id = 1; }\n"})
	if !bare.RPCs["S.Do"] || !bare.Requests["DoRequest"]["id"] {
		t.Errorf("no package: rpcs %v, requests %v", bare.RPCs, bare.Requests)
	}
	// gen/ is generated code at the top only; vendor and node_modules are
	// copies wherever they sit, and only as whole directory names.
	for p, is := range map[string]bool{
		"billing/v1/billing.proto":     true,
		"gen/billing.proto":            false,
		"billing/gen/billing.proto":    true,
		"vendor/x/a.proto":             false,
		"third/vendor/x/a.proto":       false,
		"web/node_modules/x/a.proto":   false,
		"vendors/v1/vendors.proto":     true,
		"billing/v1/billing.proto.txt": false,
	} {
		if isProto(p) != is {
			t.Errorf("isProto(%q) = %v", p, !is)
		}
	}
	fields := map[string]bool{
		"billing.v1.ListInvoicesRequest.customer": true, "billing.v1.ListInvoicesRequest.page_size": true,
		"billing.v1.ListInvoicesRequest.labels": true, "billing.v1.ListInvoicesRequest.after": true,
		"billing.v1.ListInvoicesResponse.items": true, "billing.v1.SettleRequest.key": true,
	}
	if !reflect.DeepEqual(s.Fields, fields) {
		t.Errorf("fields %v", s.Fields)
	}
}

// The servers block is read from a fence whose info string is servers, as
// many as a note has; an unclosed block or one that is not a list is refused.
func TestTheServersBlockIsReadFromItsFence(t *testing.T) {
	n := NewNote("design/changes/2026-10-24-x.md", "# X\n\n```servers\n- name: billing.v1.ListInvoicesRequest.currency\n  server: billing\n  callers: [ledger]\n```\n\n```yaml\n- name: not.this\n```\n\n```servers\n[]\n```\n")
	entries, found, err := ParseServers(n)
	if err != nil || !found || len(entries) != 1 {
		t.Fatalf("entries %v, found %v, err %v", entries, found, err)
	}
	if e := entries[0]; e.Server != "billing" || e.Callers[0] != "ledger" || e.Note != n.Path {
		t.Fatalf("got %+v", e)
	}
	// The note template shows a block in a comment, and a design may quote
	// one inside a longer fence: neither lists anything.
	shown := "# X\n\n<!-- as in:\n\n    ```servers\n    - name: an.Example\n      server: billing\n    ```\n-->\n\n````\n```servers\n- name: a.Quoted\n  server: billing\n```\n````\n"
	if entries, found, err := ParseServers(NewNote("a.md", shown)); found || len(entries) != 0 || err != nil {
		t.Fatalf("an example was read as a block: %v, %v, %v", entries, found, err)
	}
	if _, found, _ := ParseServers(NewNote("a.md", "# none\n")); found {
		t.Fatal("a note with no block was read as having one")
	}
	if _, _, err := ParseServers(NewNote("a.md", "```servers\n- name: x\n")); err == nil {
		t.Fatal("an unclosed block was accepted")
	}
	if _, _, err := ParseServers(NewNote("a.md", "```servers\nname: x\n```\n")); err == nil {
		t.Fatal("a block that is not a list was accepted")
	}
}

// A repository's role is read from the root's map: the one of kind contracts
// is the contracts repository, any other under the same root is a server, and
// one under no root, or under a root that keeps no contracts repository, has
// none and is held to nothing.
func TestARepositorysRoleIsReadFromTheRootsMap(t *testing.T) {
	root := t.TempDir()
	at := func(path string, repos []design.RepoEntry) *Env {
		return LocateEnv(&design.Parent{Root: root, Name: "project", Path: path, Design: design.Design{Repos: repos}})
	}
	if env := at("contracts", theMap); env.Role != "contracts" || env.Self != "contracts" {
		t.Errorf("the contracts repository was read as %+v", env)
	}
	if env := at("billing", theMap); env.Role != "server" || env.Self != "billing" || env.ContractsDir != filepath.Join(root, "contracts") {
		t.Errorf("a service was read as %+v", env)
	}
	if env := at("billing", []design.RepoEntry{{Path: "billing", Kind: "service"}}); env.Role != "" {
		t.Errorf("a root with no contracts repository gave a role: %+v", env)
	}
	if env := at("billing", []design.RepoEntry{{Path: "contracts", Kind: "contracts", Status: "planned"}, {Path: "billing"}}); env.Role != "" {
		t.Errorf("a contracts repository that is only planned gave a role: %+v", env)
	}
	if env := LocateEnv(nil); env.Role != "" {
		t.Errorf("a repository under no root was given a role: %+v", env)
	}
	res := &Result{}
	checkContracts(&design.Repo{Root: root, Name: "alone"}, LocateEnv(nil), res)
	if res.failed() {
		t.Errorf("a repository with no role was held to something: %v", res.Failures)
	}
}

// In the contracts repository, an RPC or request field a branch adds over
// what is merged is refused until a note lists it with a server and callers
// on the root's map; a new request message is covered by its RPC; a note the
// rules bind with no servers block is refused, and `[]` is an answer.
func TestANewDefinitionIsListedWithItsServer(t *testing.T) {
	added := strings.Replace(billingV1, "  reserved 9;", "  reserved 9;\n  string currency = 5;", 1)
	added = strings.Replace(added, "  rpc ListInvoices(", "  rpc Refund(RefundRequest) returns (RefundResponse);\n  rpc ListInvoices(", 1)
	added += "message RefundRequest { string key = 1; }\n"

	run := func(t *testing.T, notes map[string]string) *Result {
		t.Helper()
		root := mergedRepo(t, map[string]string{"billing/v1/billing.proto": billingV1})
		files := map[string]string{"billing/v1/billing.proto": added}
		for k, v := range notes {
			files[k] = v
		}
		writeFiles(t, root, files)
		repo := &design.Repo{Root: root, Name: "contracts", Files: []string{"billing/v1/billing.proto"},
			Design: design.Design{Notes: design.Notes{From: "2026-10-24"}}}
		res := &Result{}
		checkContracts(repo, &Env{Role: "contracts", Self: "contracts", Repos: theMap}, res)
		return res
	}

	t.Run("unlisted", func(t *testing.T) {
		res := run(t, map[string]string{"design/changes/2026-10-23-old.md": "# old, unbound\n"})
		got := strings.Join(res.Failures, "\n")
		for _, name := range []string{"billing.v1.BillingService.Refund", "billing.v1.ListInvoicesRequest.currency"} {
			if !strings.Contains(got, name+" is new over origin/main") {
				t.Errorf("%s not refused: %v", name, res.Failures)
			}
		}
		if len(res.Failures) != 2 || strings.Contains(got, "RefundRequest.key") {
			t.Errorf("expected exactly the RPC and the field, got %v", res.Failures)
		}
	})
	t.Run("listed", func(t *testing.T) {
		res := run(t, map[string]string{"design/changes/2026-10-24-new.md": "# new\n\n## Served by\n\n```servers\n" +
			"- name: billing.v1.BillingService.Refund\n  server: billing\n  callers: [ledger]\n" +
			"- name: billing.v1.ListInvoicesRequest.currency\n  server: billing\n  callers: [ledger]\n```\n"})
		if res.failed() {
			t.Fatalf("expected the listed names to pass, got %v", res.Failures)
		}
	})
	t.Run("off the map", func(t *testing.T) {
		res := run(t, map[string]string{"design/changes/2026-10-24-new.md": "```servers\n" +
			"- name: billing.v1.BillingService.Refund\n  server: payments\n  callers: [ledger]\n" +
			"- name: billing.v1.ListInvoicesRequest.currency\n  server: billing\n```\n"})
		got := strings.Join(res.Failures, "\n")
		if !strings.Contains(got, `"payments", which the root's map does not list`) || !strings.Contains(got, "names no caller") || len(res.Failures) != 2 {
			t.Fatalf("got %v", res.Failures)
		}
	})
	t.Run("nothing merged is here to read against", func(t *testing.T) {
		// A clone with a remote and none of its merged refs: comparing the
		// branch with itself would pass everything, so it is refused.
		root := mergedRepo(t, map[string]string{"billing/v1/billing.proto": billingV1})
		git(t, root, "remote", "add", "origin", "git@host:x/contracts.git")
		git(t, root, "update-ref", "-d", "refs/remotes/origin/main")
		writeFiles(t, root, map[string]string{"billing/v1/billing.proto": added})
		git(t, root, "commit", "-q", "-am", "feat: on a branch")
		repo := &design.Repo{Root: root, Name: "contracts", Files: []string{"billing/v1/billing.proto"}}
		res := &Result{}
		checkContracts(repo, &Env{Role: "contracts", Self: "contracts", Repos: theMap}, res)
		if len(res.Failures) != 1 || !strings.Contains(res.Failures[0], "cannot tell what this repository has merged") {
			t.Fatalf("got %v", res.Failures)
		}
	})
	t.Run("no block", func(t *testing.T) {
		res := run(t, map[string]string{
			"design/changes/2026-10-24-a.md": "# a\n",
			"design/changes/2026-10-24-b.md": "# b\n\n```servers\n- name: billing.v1.BillingService.Refund\n  server: billing\n  callers: [ledger]\n- name: billing.v1.ListInvoicesRequest.currency\n  server: billing\n  callers: [ledger]\n```\n",
			"design/changes/2026-10-24-c.md": "# c, comments only\n\n```servers\n[]\n```\n",
		})
		if len(res.Failures) != 1 || !strings.Contains(res.Failures[0], "2026-10-24-a.md: no servers block") {
			t.Fatalf("got %v", res.Failures)
		}
	})
}

// A serving repository is refused for every entry the contracts repository
// has merged naming it as server until its design's serves: names a test for
// it that exists; an entry still on a contracts branch binds nothing, nor
// does one naming another server; and a missing contracts checkout is a
// failure, not a skip.
func TestAListedDefinitionIsRefusedUntilItsServerNamesATest(t *testing.T) {
	note := "# currency\n\n```servers\n" +
		"- name: billing.v1.ListInvoicesRequest.currency\n  server: billing\n  callers: [ledger]\n" +
		"- name: ledger.v1.LedgerService.Post\n  server: ledger\n  callers: [billing]\n```\n"
	contracts := mergedRepo(t, map[string]string{
		"design/changes/2026-10-22-currency.md": note,
		"billing/v1/billing.proto":              strings.Replace(billingV1, "string customer = 1;", "string customer = 1;\n  string currency = 10;", 1),
	})
	// On a branch, unmerged: binds nothing yet.
	git(t, contracts, "checkout", "-q", "-b", "feat/next")
	writeFiles(t, contracts, map[string]string{"design/changes/2026-10-24-next.md": "```servers\n- name: billing.v1.ListInvoicesRequest.later\n  server: billing\n  callers: [ledger]\n```\n"})
	git(t, contracts, "add", "-A")
	git(t, contracts, "commit", "-q", "-m", "next")
	env := &Env{Role: "server", Self: "billing", ContractsDir: contracts, Repos: theMap}

	server := func(serves []design.Served, test string) *design.Repo {
		root := t.TempDir()
		writeFiles(t, root, map[string]string{"internal/invoice/list_test.go": test})
		return &design.Repo{Root: root, Name: "billing", Files: []string{"internal/invoice/list_test.go"},
			Design: design.Design{Files: []string{"internal/**"}, Serves: serves}}
	}
	const field = "billing.v1.ListInvoicesRequest.currency"

	res := &Result{}
	checkContracts(server(nil, "package invoice\n"), env, res)
	if len(res.Failures) != 1 || !strings.Contains(res.Failures[0], field+": contracts (design/changes/2026-10-22-currency.md, at origin/main) says billing serves it, called by ledger") {
		t.Fatalf("expected the unserved field alone to be refused, got %v", res.Failures)
	}

	res = &Result{}
	checkContracts(server([]design.Served{{Name: field, Test: "TestListNarrowsToOneCurrency"}}, "package invoice\n"), env, res)
	if len(res.Failures) != 1 || !strings.Contains(res.Failures[0], "TestListNarrowsToOneCurrency, which is in no test file") {
		t.Fatalf("expected the missing test to be refused, got %v", res.Failures)
	}

	res = &Result{}
	checkContracts(server([]design.Served{{Name: field, Test: "TestListNarrowsToOneCurrency"}},
		"package invoice\nfunc TestListNarrowsToOneCurrency(t *testing.T) {}\n"), env, res)
	if res.failed() {
		t.Fatalf("expected a served field with its test to pass, got %v", res.Failures)
	}

	res = &Result{}
	checkContracts(server(nil, ""), &Env{Role: "server", Self: "billing", ContractsDir: filepath.Join(t.TempDir(), "contracts"), Repos: theMap}, res)
	if len(res.Failures) != 1 || !strings.Contains(res.Failures[0], "the contracts repository is not checked out") {
		t.Fatalf("expected a missing contracts checkout to fail, got %v", res.Failures)
	}
}

// A note may name what no merged definition declares yet: that is a design,
// and it binds no server. The definition merging binds it.
func TestAPlannedEntryBindsNoServerUntilItsDefinitionMerges(t *testing.T) {
	note := "# posting\n\n```servers\n" +
		"- name: ledger.v1.LedgerService.Post\n  server: ledger\n  callers: [billing]\n" +
		"- name: ledger.v1.Entry.memo\n  server: ledger\n  callers: [billing]\n```\n"
	ledger := func() *design.Repo {
		return &design.Repo{Root: t.TempDir(), Name: "ledger"}
	}
	env := func(dir string) *Env {
		return &Env{Role: "server", Self: "ledger", ContractsDir: dir, Repos: theMap}
	}

	planned := mergedRepo(t, map[string]string{"design/changes/2026-10-25-posting.md": note})
	res := &Result{}
	checkContracts(ledger(), env(planned), res)
	if res.failed() {
		t.Fatalf("a design with no definition bound its server: %v", res.Failures)
	}

	built := mergedRepo(t, map[string]string{
		"design/changes/2026-10-25-posting.md": note,
		"ledger/v1/ledger.proto":               "syntax = \"proto3\";\npackage ledger.v1;\nservice LedgerService {\n  rpc Post(PostRequest) returns (PostResponse);\n}\nmessage Entry {\n  string memo = 6;\n}\n",
	})
	res = &Result{}
	checkContracts(ledger(), env(built), res)
	got := strings.Join(res.Failures, "\n")
	if len(res.Failures) != 2 || !strings.Contains(got, "ledger.v1.LedgerService.Post: contracts") || !strings.Contains(got, "ledger.v1.Entry.memo: contracts") {
		t.Fatalf("expected both merged names to bind ledger, got %v", res.Failures)
	}
}

// Where no entry of the root's map is the contracts repository and the signs
// of one are there, a serves: key, a servers block in a note or .proto
// files, the checker says in a line that the contracts rules are off, and
// still passes. A map that gives the kind, even to a repository only
// planned, a repository with none of the signs, and one under no root are
// told nothing.
func TestTheContractsRulesSayWhenTheyAreOff(t *testing.T) {
	otherKind := []design.RepoEntry{{Path: "protos", Kind: "definitions"}, {Path: "billing", Kind: "service"}}
	under := func(repos []design.RepoEntry, path string) *design.Parent {
		return &design.Parent{Root: t.TempDir(), Name: "project", Path: path, Design: design.Design{Repos: repos}}
	}
	nested := func(files map[string]string, serves []design.Served) *design.Repo {
		root := t.TempDir()
		writeFiles(t, root, files)
		var names []string
		for f := range files {
			if !strings.HasPrefix(f, "design/") {
				names = append(names, f)
			}
		}
		return &design.Repo{Root: root, Name: "billing", Files: names, Design: design.Design{Files: []string{"**"}, Serves: serves}}
	}
	served := []design.Served{{Name: "billing.v1.BillingService.Settle", Test: "TestSettle"}}
	block := map[string]string{"design/changes/2026-10-22-currency.md": "# currency\n\n```servers\n[]\n```\n"}
	protos := map[string]string{"billing/v1/billing.proto": billingV1, "gen/billing.proto": billingV1}

	cases := []struct {
		name   string
		repo   *design.Repo
		parent *design.Parent
		says   []string // nil when nothing is said
	}{
		{"a design with serves:", nested(nil, served), under(otherKind, "billing"), []string{"this design has serves:", "the map of project"}},
		{"a note with a servers block", nested(block, nil), under(otherKind, "protos"), []string{"a note in this repository carries a servers block (design/changes/2026-10-22-currency.md)"}},
		{"definitions", nested(protos, nil), under(otherKind, "protos"), []string{"this repository holds .proto files"}},
		{"every sign at once", nested(protos, served), under(otherKind, "protos"), []string{"this design has serves:; this repository holds .proto files"}},
		{"only generated definitions", nested(map[string]string{"gen/billing.proto": billingV1}, nil), under(otherKind, "billing"), nil},
		{"no sign", nested(map[string]string{"main.go": "package main\n"}, nil), under(otherKind, "billing"), nil},
		{"the map gives the kind", nested(nil, served), under(theMap, "billing"), nil},
		{"the map gives the kind to one only planned", nested(nil, served), under([]design.RepoEntry{{Path: "contracts", Kind: "contracts", Status: "planned"}, {Path: "billing"}}, "billing"), nil},
		{"under no root", nested(protos, served), nil, nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := contractsOff(c.repo, c.parent, LocateEnv(c.parent))
			if c.says == nil {
				if got != "" {
					t.Fatalf("nothing was to be said, got: %s", got)
				}
				return
			}
			for _, want := range append(c.says, "the contracts rules are off", "`kind: contracts`") {
				if !strings.Contains(got, want) {
					t.Errorf("expected %q in: %s", want, got)
				}
			}
		})
	}

	// At the root, the signs are looked for in each repository on its map,
	// and the line is printed with a run that passes.
	root := t.TempDir()
	writeFiles(t, root, map[string]string{
		"protos/billing/v1/billing.proto": billingV1,
		"billing/design/DESIGN.md":        "---\nfiles: [main.go]\nserves:\n  - name: billing.v1.BillingService.Settle\n    test: TestSettle\n---\n\n# billing\n",
		"billing/main.go":                 "package main\n",
	})
	atRoot := &design.Repo{Root: root, Name: "project", Design: design.Design{Repos: otherKind}}
	res := runChecks(atRoot)
	var out strings.Builder
	res.report(&out, atRoot)
	for _, want := range []string{"harness: the contracts rules are off: protos holds .proto files; billing's design has serves:, and design/DESIGN.md repos: gives no repository `kind: contracts`", "harness: OK"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("at the root, expected %q in:\n%s", want, out.String())
		}
	}
	if res.failed() {
		t.Errorf("the line is not a refusal, yet the run failed: %v", res.Failures)
	}
	atRoot.Design.Repos = theMap
	if res := runChecks(atRoot); len(res.Notices) != 0 {
		t.Errorf("a root whose map gives the kind was told the rules are off: %v", res.Notices)
	}
}

// A test that serves: names must exist whatever the root's map says: in a
// repository with no role, under a root with no contracts repository or
// under no root at all, a missing one is refused and one that is there
// passes.
func TestATestServesNamesExistsWhateverTheMapSays(t *testing.T) {
	served := []design.Served{{Name: "billing.v1.BillingService.Settle", Test: "TestSettleIsIdempotent"}}
	server := func(test string) *design.Repo {
		root := t.TempDir()
		writeFiles(t, root, map[string]string{"internal/settle_test.go": test})
		return &design.Repo{Root: root, Name: "billing", Files: []string{"internal/settle_test.go"},
			Design: design.Design{Files: []string{"internal/**"}, Serves: served}}
	}
	noContracts := &design.Parent{Root: t.TempDir(), Name: "project", Path: "billing",
		Design: design.Design{Repos: []design.RepoEntry{{Path: "protos", Kind: "definitions"}, {Path: "billing", Kind: "service"}}}}
	for _, parent := range []*design.Parent{nil, noContracts} {
		env := LocateEnv(parent)
		if env.Role != "" {
			t.Fatalf("the fixture has a role: %+v", env)
		}
		res := &Result{}
		checkContracts(server("package internal\n"), env, res)
		if len(res.Failures) != 1 || !strings.Contains(res.Failures[0], "TestSettleIsIdempotent, which is in no test file") {
			t.Errorf("expected the missing test to be refused, got %v", res.Failures)
		}
		res = &Result{}
		checkContracts(server("package internal\nfunc TestSettleIsIdempotent(t *testing.T) {}\n"), env, res)
		if res.failed() {
			t.Errorf("expected a test that exists to pass, got %v", res.Failures)
		}
	}
}
