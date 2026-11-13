// Package finder picks existing tabs and idle shells. It only looks at data
// (hc terminals, herdr-app ls), so it is easy to test.
package finder

import (
	"github.com/shu-go/hccd/internal/hc"
	"github.com/shu-go/hccd/internal/herdr"
	"github.com/shu-go/hccd/internal/target"
)

// Candidates returns the local live terminals shown in a Herdr tab whose
// shell is now in dir.
func Candidates(terms []hc.Terminal, dir string) []hc.Terminal {
	var out []hc.Terminal
	for _, t := range terms {
		if t.State != "live" || !t.Local() || t.Tab() == "" {
			continue
		}
		if target.SamePath(t.CurrentDir(), dir) {
			out = append(out, t)
		}
	}
	return out
}

// Tab picks the tab for dir: the tab being shown if it matches, else the
// first matching tab in herdr-app ls order. "" if none.
func Tab(terms []hc.Terminal, ls *herdr.Ls, dir string) string {
	match := map[string]bool{}
	for _, t := range Candidates(terms, dir) {
		match[t.Tab()] = true
	}
	if len(match) == 0 {
		return ""
	}
	if a := ls.ActiveTab(); match[a] {
		return a
	}
	for _, s := range ls.Spaces {
		for _, t := range s.Tabs {
			if match[t.ID] {
				return t.ID
			}
		}
	}
	return "" // matching terminals are not in a tab the window knows
}

// Busy reports whether a shell terminal is doing something: a program that
// reports status (an agent) is in it, or the shell has child processes.
type Busy func(t hc.Terminal) bool

// IdleShell returns a terminal in tab whose shell is in dir and not busy.
func IdleShell(terms []hc.Terminal, tab, dir string, busy Busy) (hc.Terminal, bool) {
	for _, t := range Candidates(terms, dir) {
		if t.Tab() == tab && !busy(t) {
			return t, true
		}
	}
	return hc.Terminal{}, false
}
