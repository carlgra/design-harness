package main

import (
	"sort"
	"strings"

	"harness/internal/design"
	"harness/internal/history"
)

// own is the name the files no area holds go by: the design's own.
const own = ""

// AreaScore is how the design's division into areas fits the history.
type AreaScore struct {
	Defined int // areas the design defines
	// Code is the number of changes that touched a file the design accounts
	// for. The rest touched documents only and are not counted.
	Code int
	// By is the number of those changes that touched no area, one, two, and
	// three or more.
	By [4]int
	// Mixed is, of the changes inside one area, those that also touched
	// files no area holds.
	Mixed int
	// Sweeps are changes that touched more than sweep areas: a rename or a
	// reformat, counted in By and left out of Pairs.
	Sweeps int
	Areas  []AreaLine
	Pairs  []history.Pair
	// Loose are the directories no area holds that changed most, each with
	// the areas it changed with: where the next area is, or where an
	// existing one's boundary should be.
	Loose []LooseDir
	// Suggest are directories that change together, where the design
	// defines no area at all.
	// Examples are the most recent changes that touched both areas of a
	// pair: what to read to see why they move together.
	Examples map[[2]string][]history.Commit
	Suggest  []history.Pair
	Dirs     map[string]int
}

// AreaLine is one area over the history.
type AreaLine struct {
	Name    string
	Changes int // changes that touched it
	Alone   int // of those, the ones that touched no other area
}

// LooseDir is a directory outside every area.
type LooseDir struct {
	Dir     string
	Changes int
	Alone   int // changes that touched it and no area
	// With are the areas it changed with most, and how often.
	With []Count
}

// Count is a name and how often.
type Count struct {
	Name string
	N    int
}

// sweep is how many areas, or directories, a change may touch and still say
// something about which belong together.
const sweep = 8

// scoreAreas sets the areas of the design against the changes. A file is
// placed by the design as it stands today, whatever the design said when the
// change was made: the question is whether today's boundaries would have
// held.
func scoreAreas(repo *design.Repo, commits []history.Commit) *AreaScore {
	s := &AreaScore{Defined: len(repo.Design.Spines), Dirs: map[string]int{}, Examples: map[[2]string][]history.Commit{}}
	area := func(p string) (string, bool) {
		if strings.HasPrefix(p, "design/") || design.Exempt[p] {
			return "", false
		}
		if a := repo.AreasOf(p); len(a) > 0 {
			return a[0], true
		}
		return own, true
	}
	lines := map[string]*AreaLine{}
	for _, a := range repo.Design.Spines {
		lines[a.Name] = &AreaLine{Name: a.Name}
	}
	loose := map[string]int{}
	looseAlone := map[string]int{}
	looseWith := map[string]map[string]int{}
	var coded []history.Commit
	for _, c := range commits {
		areas := map[string]bool{}
		dirs := map[string]bool{}
		code := false
		for _, p := range c.Paths() {
			a, ok := area(p)
			if !ok {
				continue
			}
			code = true
			if a == own {
				dirs[history.Dir(p)] = true
			} else {
				areas[a] = true
			}
		}
		if !code {
			continue
		}
		coded = append(coded, c)
		s.Code++
		n := len(areas)
		if n > 3 {
			n = 3
		}
		s.By[n]++
		if len(areas) == 1 && len(dirs) > 0 {
			s.Mixed++
		}
		if len(areas) > sweep {
			s.Sweeps++
		}
		for a := range areas {
			lines[a].Changes++
			if len(areas) == 1 {
				lines[a].Alone++
			}
		}
		if len(dirs) > sweep {
			continue
		}
		if len(areas) == 2 {
			var two []string
			for a := range areas {
				two = append(two, a)
			}
			sort.Strings(two)
			k := [2]string{two[0], two[1]}
			if len(s.Examples[k]) < 2 {
				s.Examples[k] = append(s.Examples[k], c)
			}
		}
		for d := range dirs {
			if len(areas) == 0 {
				looseAlone[d]++
			}
			loose[d]++
			if looseWith[d] == nil {
				looseWith[d] = map[string]int{}
			}
			for a := range areas {
				looseWith[d][a]++
			}
		}
	}
	for _, a := range repo.Design.Spines {
		s.Areas = append(s.Areas, *lines[a.Name])
	}
	if s.Defined > 0 {
		s.Pairs, _ = history.Together(coded, func(p string) string {
			if a, ok := area(p); ok && a != own {
				return a
			}
			return ""
		}, sweep)
		for d, n := range loose {
			l := LooseDir{Dir: d, Changes: n, Alone: looseAlone[d]}
			for a, m := range looseWith[d] {
				l.With = append(l.With, Count{a, m})
			}
			sort.Slice(l.With, func(i, j int) bool {
				if l.With[i].N != l.With[j].N {
					return l.With[i].N > l.With[j].N
				}
				return l.With[i].Name < l.With[j].Name
			})
			s.Loose = append(s.Loose, l)
		}
		sort.Slice(s.Loose, func(i, j int) bool {
			if s.Loose[i].Changes != s.Loose[j].Changes {
				return s.Loose[i].Changes > s.Loose[j].Changes
			}
			return s.Loose[i].Dir < s.Loose[j].Dir
		})
		return s
	}
	s.Suggest, s.Dirs = history.Together(coded, func(p string) string {
		if _, ok := area(p); ok {
			return history.Dir(p)
		}
		return ""
	}, sweep)
	return s
}
