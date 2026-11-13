package finder

import (
	"encoding/json"
	"runtime"
	"testing"

	"github.com/shu-go/hccd/internal/hc"
	"github.com/shu-go/hccd/internal/herdr"
)

func term(addr, tab, fgCwd, status string) hc.Terminal {
	t := hc.Terminal{
		Address:    addr,
		State:      "live",
		Labels:     map[string]string{"herdr.tab": tab, "herdr.space": "sp1"},
		Foreground: &hc.Foreground{Name: "pwsh.exe", Cwd: fgCwd, Pid: 1},
		Status:     json.RawMessage(status),
	}
	if tab == "" {
		t.Labels = map[string]string{}
	}
	return t
}

func dirs() (a, b string) {
	if runtime.GOOS == "windows" {
		return `C:\work\a`, `C:\work\b`
	}
	return "/work/a", "/work/b"
}

func lsOf(active string, tabs ...string) *herdr.Ls {
	s := herdr.Space{ID: "sp1", Active: true}
	for _, id := range tabs {
		s.Tabs = append(s.Tabs, herdr.Tab{ID: id, Active: id == active})
	}
	return &herdr.Ls{Spaces: []herdr.Space{s}}
}

func TestTab(t *testing.T) {
	a, b := dirs()
	terms := []hc.Terminal{
		term("t1", "tb1", b, "null"),
		term("t2", "tb2", a, "null"),
		term("t3", "tb3", a, "null"),
		term("t4", "", a, "null"),         // hidden worker: no tab
		{Address: "vps/t5", State: "live", // remote
			Labels: map[string]string{"herdr.tab": "tb5"}, Foreground: &hc.Foreground{Cwd: a}},
	}
	if got := Tab(terms, lsOf("tb1", "tb1", "tb2", "tb3"), a); got != "tb2" {
		t.Errorf("first in ls order: got %q", got)
	}
	if got := Tab(terms, lsOf("tb3", "tb1", "tb2", "tb3"), a); got != "tb3" {
		t.Errorf("active tab wins: got %q", got)
	}
	if got := Tab(terms, lsOf("tb1", "tb3", "tb2", "tb1"), a); got != "tb3" {
		t.Errorf("ls order, not terminal order: got %q", got)
	}
	if got := Tab(terms[:1], lsOf("tb1", "tb1"), a); got != "" {
		t.Errorf("no match: got %q", got)
	}
	if got := Tab(terms, lsOf("tb1", "tb1"), a); got != "" {
		t.Errorf("matching tabs unknown to the window: got %q", got)
	}
}

func TestTabUsesCurrentDirNotStartDir(t *testing.T) {
	a, b := dirs()
	x := term("t1", "tb1", b, "null")
	x.Cwd = a // started in a, then cd b
	if got := Tab([]hc.Terminal{x}, lsOf("tb1", "tb1"), a); got != "" {
		t.Errorf("start dir must not match: got %q", got)
	}
}

func TestIdleShell(t *testing.T) {
	a, _ := dirs()
	agent := term("t1", "tb1", a, `{"app":"claude-code","state":"working"}`)
	shell := term("t2", "tb1", a, "null")
	other := term("t3", "tb2", a, "null")
	busy := func(t hc.Terminal) bool { return t.HasStatus() }

	got, ok := IdleShell([]hc.Terminal{agent, shell, other}, "tb1", a, busy)
	if !ok || got.Address != "t2" {
		t.Errorf("got %v %v, want t2", got.Address, ok)
	}
	if _, ok := IdleShell([]hc.Terminal{agent, other}, "tb1", a, busy); ok {
		t.Error("only agent in tab: want none")
	}
}
