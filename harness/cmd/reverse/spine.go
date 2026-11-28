package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"harness/internal/design"
)

// SpineHeader opens a drafting session's reply when what it drafts is the
// spine of one area.
func SpineHeader(area string) string { return "=== " + design.SpinePath(area) + " ===" }

// areaOf is the area called name as the design defines it.
func areaOf(d design.Design, name string) (design.Area, error) {
	for _, a := range d.Spines {
		if a.Name == name {
			if len(a.Files) == 0 {
				return a, fmt.Errorf("the area %s lists no files: in %s", name, design.DocPath)
			}
			return a, nil
		}
	}
	return design.Area{}, fmt.Errorf("%s defines no area called %s: add it under spines:, with the files: it holds, and then write its spine", design.DocPath, name)
}

// areaSurvey surveys one area: the files the design gives it and nothing
// else, so that what a session is handed about a part of a large repository
// is the size of the part.
func areaSurvey(repo *design.Repo, area design.Area, today string) Survey {
	part := &design.Repo{Root: repo.Root, Name: area.Name}
	for _, f := range repo.Files {
		if design.Claimed(area.Files, f) {
			part.Files = append(part.Files, f)
		}
	}
	s := survey(part, nil, nil, today)
	s.Globs = area.Files
	// The walk leaves design/ out and the survey adds the repository's own
	// design back; here that is handed over whole, not listed.
	var docs []string
	for _, d := range s.Docs {
		if !strings.HasPrefix(d, "design/") {
			docs = append(docs, d)
		}
	}
	s.Docs, s.Gates, s.Harness, s.OwnHarness = docs, nil, 0, false
	// Documents outside the area may still hold its rules: an architecture
	// document written for the whole repository, a decision record. They are
	// named, so that what they say of this area is brought into its spine.
	_, err := os.Stat(filepath.Join(repo.Root, "harness", "FRAMEWORK.md"))
	for _, f := range repo.Files {
		if design.Claimed(area.Files, f) || (err == nil && strings.HasPrefix(f, "harness/")) {
			continue
		}
		if isDoc(strings.ToLower(f)) {
			s.Elsewhere = append(s.Elsewhere, f)
		}
	}
	return s
}

// AreaMarkdown is the survey of one area as a session, or a person, is
// handed it.
func (s Survey) AreaMarkdown(repo string) string {
	b := &strings.Builder{}
	fmt.Fprintf(b, "# Survey of the area %s, in %s\n\nRead by a machine on %s. Facts only: nothing here is a decision.\n\n", s.Name, repo, s.Date)
	fmt.Fprintf(b, "## Files\n\n%d files, under the globs `design/DESIGN.md` gives the area:\n\n", s.Files)
	list(b, s.Globs, "`%s`")
	b.WriteString("\n## Languages\n\n")
	if len(s.Languages) == 0 {
		b.WriteString("None recognised.\n")
	}
	for _, l := range s.Languages {
		fmt.Fprintf(b, "- %s: %d file(s)\n", l.Name, l.Files)
	}
	fmt.Fprintf(b, "\n## Tests\n\n%d test file(s), %d test name(s) found in the area.\n", len(s.TestFiles), len(s.Tests))
	if len(s.TestFiles) > 0 {
		b.WriteString("\nTest files:\n\n")
		list(b, s.TestFiles, "`%s`")
	}
	if len(s.Tests) > 0 {
		b.WriteString("\nTest names, the only ones the spine's `tests:` may name:\n\n")
		list(b, s.Tests, "`%s`")
	}
	b.WriteString("\n## Documents in the area that may already record what was decided\n\n")
	if len(s.Docs) == 0 {
		b.WriteString("None found.\n")
	}
	list(b, s.Docs, "`%s`")
	if len(s.Elsewhere) > 0 {
		b.WriteString("\n## Documents elsewhere in the repository that may hold rules of this area\n\nWritten for the repository as a whole, or for other parts. What they say that governs this area alone belongs in this spine.\n\n")
		list(b, s.Elsewhere, "`%s`")
	}
	return b.String()
}

// SpineSkeleton fills the spine template: a document the checker reads at
// once, and a body whose every section is still a question. Its placeholders
// are {{AREA}}, {{REPO}} and {{FILES}}.
func (s Survey) SpineSkeleton(template, repo string) string {
	files := &strings.Builder{}
	for _, g := range s.Globs {
		fmt.Fprintf(files, "- `%s`\n", g)
	}
	return strings.NewReplacer(
		"{{AREA}}", s.Name,
		"{{REPO}}", repo,
		"{{FILES}}", strings.TrimRight(files.String(), "\n"),
	).Replace(template)
}

// spinePrompt is the whole of what a session drafting a spine is given: the
// instructions, the survey of the area, the shape to fill, and the
// repository's design. No other area's spine is in it.
func spinePrompt(harness string, s Survey, repo, template, mainDoc string) (string, error) {
	instructions, err := os.ReadFile(filepath.Join(harness, "reverse", "spine.md"))
	if err != nil {
		return "", err
	}
	b := &strings.Builder{}
	b.WriteString(strings.TrimRight(strings.ReplaceAll(string(instructions), "{{HEADER}}", SpineHeader(s.Name)), "\n"))
	b.WriteString("\n\n---\n\n")
	b.WriteString(strings.TrimSpace(s.AreaMarkdown(repo)))
	b.WriteString(statedRules(s.Stated))
	b.WriteString("\n\n---\n\nThe shape to fill:\n\n````markdown\n")
	b.WriteString(strings.TrimSpace(strings.ReplaceAll(s.SpineSkeleton(template, repo), SkeletonMark+"\n\n", "")))
	fmt.Fprintf(b, "\n````\n\n---\n\nThe design of `%s`, the repository. What it decides binds this area: cite it as `%s/ADR-NNNN`, and do not restate it.\n\n````markdown\n", repo, repo)
	b.WriteString(strings.TrimSpace(mainDoc))
	b.WriteString("\n````\n")
	return b.String(), nil
}

// splitSpine reads a session's reply as the spine of area: the document
// after its header, which must be one the checker can load.
func splitSpine(reply, area string) (string, error) {
	reply = strings.ReplaceAll(reply, "\r\n", "\n")
	header := SpineHeader(area)
	i := strings.Index(reply, header)
	if i < 0 {
		return "", fmt.Errorf("the reply has no %q line", header)
	}
	doc := strings.TrimSpace(unfence(reply[i+len(header):])) + "\n"
	fm, err := design.ParseFrontmatter(doc)
	if err != nil {
		return "", fmt.Errorf("the spine in the reply: %w", err)
	}
	var head struct {
		Tests []string `yaml:"tests"`
	}
	if err := yaml.Unmarshal([]byte(fm.YAML), &head); err != nil {
		return "", fmt.Errorf("the spine in the reply: frontmatter: %w", err)
	}
	if strings.TrimSpace(fm.Body) == "" {
		return "", errors.New("the spine in the reply is frontmatter with no document under it")
	}
	return doc, nil
}

// spineStep is a step of the command asked of one area and not of the whole
// repository: skeleton, prompt or draft.
func spineStep(step, root, harness, area string, force bool, repo *design.Repo, today string, ask func(model, prompt string) (string, error), model, stated string) error {
	full, err := design.Load(root)
	if err != nil {
		return fmt.Errorf("a spine is written under a design: %w", err)
	}
	a, err := areaOf(full.Design, area)
	if err != nil {
		return err
	}
	template, err := os.ReadFile(filepath.Join(harness, "templates", "spine.md"))
	if err != nil {
		return err
	}
	s := areaSurvey(repo, a, today)
	rel := design.SpinePath(area)
	path := filepath.Join(root, filepath.FromSlash(rel))
	mainDoc, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(design.DocPath)))
	if err != nil {
		return err
	}
	hp := "harness"
	if rp, err := filepath.Rel(root, harness); err == nil && !strings.HasPrefix(rp, ".."+string(filepath.Separator)+".."+string(filepath.Separator)+"..") {
		hp = filepath.ToSlash(rp)
	}
	switch step {
	case "skeleton":
		if _, err := os.Stat(path); err == nil {
			return fmt.Errorf("%s already exists; a skeleton is only ever the first one", rel)
		}
		if err := write(path, s.SpineSkeleton(string(template), full.Name)); err != nil {
			return err
		}
		fmt.Printf("reverse: wrote %s: the spine of %d files, every section still a question\n", rel, s.Files)
		fmt.Printf("reverse: next, draft it:  go -C %s run ./cmd/reverse draft -spine %s\n", hp, area)
	case "prompt":
		p, err := spinePrompt(harness, s, full.Name, string(template), string(mainDoc))
		if err != nil {
			return err
		}
		fmt.Print(p)
	case "draft":
		if err := replaceableAt(path, rel, force); err != nil {
			return err
		}
		p, err := spinePrompt(harness, s, full.Name, string(template), string(mainDoc))
		if err != nil {
			return err
		}
		fmt.Printf("reverse: %s, the area %s: one session, model %s, reading %d files and writing none\n", full.Name, area, model, s.Files)
		reply, err := ask(model, p)
		if err != nil {
			return err
		}
		reply, took := cutTaken(reply)
		doc, err := splitSpine(reply, area)
		if err != nil {
			kept := filepath.Join(root, "design", "reverse-reply.md")
			if werr := write(kept, reply); werr == nil {
				err = fmt.Errorf("%w; the reply is kept in %s", err, design.Rel(root, kept))
			}
			return err
		}
		if err := write(path, doc); err != nil {
			return err
		}
		fmt.Printf("reverse: wrote %s\n", rel)
		if err := keepTaken(root, rel, took); err != nil {
			return err
		}
		fmt.Printf("reverse: %d place(s) are marked (inferred) and %d TODO(design): a person answers each.\n",
			strings.Count(doc, "(inferred)"), design.Marks(doc))
		fmt.Println("reverse: next:")
		fmt.Printf("    go -C %s run ./cmd/check -validate=false\n", hp)
		fmt.Printf("    go -C %s run ./cmd/coldread -spine %s -only comprehension\n", hp, area)
	default:
		return fmt.Errorf("-spine goes with skeleton, prompt or draft, not %s", step)
	}
	return nil
}

// skeletonsForAreas writes the skeleton of a spine for every area the design
// defines that has none, and says which it wrote. One that exists is left
// alone.
func skeletonsForAreas(root, harness string, repo *design.Repo, today string) ([]string, error) {
	full, err := design.Load(root)
	if err != nil {
		return nil, err
	}
	var wrote []string
	for _, a := range full.Design.Spines {
		path := filepath.Join(root, filepath.FromSlash(design.SpinePath(a.Name)))
		if _, err := os.Stat(path); err == nil || len(a.Files) == 0 || strings.ContainsAny(a.Name, "/\\ ") {
			continue
		}
		template, err := os.ReadFile(filepath.Join(harness, "templates", "spine.md"))
		if err != nil {
			return wrote, err
		}
		if err := write(path, areaSurvey(repo, a, today).SpineSkeleton(string(template), full.Name)); err != nil {
			return wrote, err
		}
		wrote = append(wrote, a.Name)
	}
	return wrote, nil
}
