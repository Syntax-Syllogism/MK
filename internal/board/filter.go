package board

import (
	"slices"
	"strings"

	"github.com/sahilm/fuzzy"
)

// Filter combines selected chips with OR within groups and AND across groups.
type Filter struct {
	Projects map[string]bool
	Epics    map[string]bool // NoEpic matches an empty Epic.
	Query    string
	Tags     map[string]bool
}

func selectedCount(group map[string]bool) int {
	n := 0
	for _, v := range group {
		if v {
			n++
		}
	}
	return n
}
func (f Filter) Active() int {
	n := selectedCount(f.Projects) + selectedCount(f.Epics) + selectedCount(f.Tags)
	if strings.TrimSpace(f.Query) != "" {
		n++
	}
	return n
}
func (f Filter) Visible(t Task) bool {
	if selectedCount(f.Projects) > 0 && (t.Project == "" || !f.Projects[t.Project]) {
		return false
	}
	epic := t.Epic
	if epic == "" {
		epic = NoEpic
	}
	if selectedCount(f.Epics) > 0 && !f.Epics[epic] {
		return false
	}
	if selectedCount(f.Tags) > 0 {
		found := false
		for _, tag := range t.Tags {
			if f.Tags[tag] {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	if strings.TrimSpace(f.Query) == "" {
		return true
	}
	return f.MatchesQuery(t, strings.ToLower(t.Content))
}

// MatchesQuery accepts a pre-lowered body so interactive consumers can cache it.
func (f Filter) MatchesQuery(t Task, lowerBody string) bool {
	for _, term := range strings.Fields(f.Query) {
		if len(fuzzy.Find(term, []string{t.Title, t.Heading})) > 0 {
			continue
		}
		needle := strings.ToLower(term)
		fields := append([]string{t.Project, t.Epic}, t.Tags...)
		fields = append(fields, t.DependsOn...)
		matched := strings.Contains(lowerBody, needle)
		for _, field := range fields {
			matched = matched || strings.Contains(strings.ToLower(field), needle)
		}
		if !matched {
			return false
		}
	}
	return true
}

// Score ranks each term by its best filename or heading match; metadata-only matches score zero.
func (f Filter) Score(t Task) int {
	score := 0
	for _, term := range strings.Fields(f.Query) {
		if matches := fuzzy.Find(term, []string{t.Title, t.Heading}); len(matches) > 0 {
			score += matches[0].Score
		}
	}
	return score
}

func Facets(tasks []Task) (projects, epics, tags []string) {
	projectSet, epicSet, tagSet := map[string]bool{}, map[string]bool{}, map[string]bool{}
	for _, t := range tasks {
		if t.Project != "" {
			projectSet[t.Project] = true
		}
		epic := t.Epic
		if epic == "" {
			epic = NoEpic
		}
		epicSet[epic] = true
		for _, tag := range t.Tags {
			tagSet[tag] = true
		}
	}
	return sortedKeys(projectSet), sortedKeys(epicSet), sortedKeys(tagSet)
}
func sortedKeys(set map[string]bool) []string {
	var keys []string
	for k := range set {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}
