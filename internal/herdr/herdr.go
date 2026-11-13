// Package herdr drives the Herdr window through the herdr-app CLI.
package herdr

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/shu-go/hccd/internal/sys"
)

type Terminal struct {
	Terminal string `json:"terminal"`
	Pane     string `json:"pane"`
}

type Tab struct {
	ID        string     `json:"id"`
	Name      string     `json:"name"`
	Active    bool       `json:"active"`
	Settled   bool       `json:"settled"`
	Terminals []Terminal `json:"terminals"`
}

type Space struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Folder string `json:"folder"`
	Active bool   `json:"active"`
	Tabs   []Tab  `json:"tabs"`
}

type Ls struct {
	Spaces []Space `json:"spaces"`
}

// ActiveTab returns the id of the tab being shown ("" if none).
func (ls *Ls) ActiveTab() string {
	for _, s := range ls.Spaces {
		if !s.Active {
			continue
		}
		for _, t := range s.Tabs {
			if t.Active {
				return t.ID
			}
		}
	}
	return ""
}

// ErrNoWindow means no Herdr window is running.
var ErrNoWindow = errors.New("no Herdr window")

func run(args ...string) ([]byte, error) {
	cmd := sys.Command("herdr-app", args...)
	var out, errb bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errb
	err := cmd.Run()
	if err != nil {
		msg := strings.TrimSpace(errb.String() + out.String())
		if strings.Contains(msg, "no Herdr window") {
			return nil, fmt.Errorf("%w: %s", ErrNoWindow, msg)
		}
		return out.Bytes(), fmt.Errorf("herdr-app %s: %w: %s", strings.Join(args, " "), err, msg)
	}
	return out.Bytes(), nil
}

func List() (*Ls, error) {
	b, err := run("ls")
	if err != nil {
		return nil, err
	}
	var ls Ls
	if err := json.Unmarshal(b, &ls); err != nil {
		return nil, fmt.Errorf("herdr-app ls: %w", err)
	}
	return &ls, nil
}

// Created is the reply of space.new / tab.new.
type Created struct {
	Space    string `json:"space"`
	Tab      string `json:"tab"`
	Terminal string `json:"terminal"`
	Error    string `json:"error"`
}

func Do(action string, args map[string]any, reply any) error {
	j, err := json.Marshal(args)
	if err != nil {
		return err
	}
	b, err := run("do", action, string(j))
	if err != nil {
		return err
	}
	if reply != nil && len(bytes.TrimSpace(b)) > 0 {
		if err := json.Unmarshal(b, reply); err != nil {
			return fmt.Errorf("herdr-app do %s: %w: %s", action, err, b)
		}
	}
	return nil
}

func NewTab(folder, name, space string) (*Created, error) {
	args := map[string]any{"folder": folder}
	if name != "" {
		args["name"] = name
	}
	if space != "" {
		args["space"] = space
	}
	var c Created
	err := Do("tab.new", args, &c)
	return &c, err
}

func NewSpace(folder, name string) (*Created, error) {
	args := map[string]any{"folder": folder}
	if name != "" {
		args["name"] = name
	}
	var c Created
	err := Do("space.new", args, &c)
	return &c, err
}

// SelectTab shows a tab; it also switches spaces.
func SelectTab(id string) error {
	return Do("tab.select", map[string]any{"id": id}, nil)
}

// Raise opens or raises the window. herdr-app is a GUI program: start it
// plainly (no SysProcAttr; HideWindow would keep its window hidden).
func Raise() error {
	cmd := exec.Command("herdr-app")
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start herdr-app: %w", err)
	}
	go cmd.Wait()
	return nil
}

// EnsureWindow starts Herdr when no window is open and waits until it
// answers. started reports whether it had to start it.
func EnsureWindow(timeout time.Duration) (started bool, err error) {
	_, err = run("status")
	if err == nil {
		return false, nil
	}
	if !errors.Is(err, ErrNoWindow) {
		return false, err
	}
	if err := Raise(); err != nil {
		return false, err
	}
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		time.Sleep(250 * time.Millisecond)
		if _, err = run("status"); err == nil {
			return true, nil
		}
	}
	return true, fmt.Errorf("Herdr did not open within %v: %w", timeout, err)
}
