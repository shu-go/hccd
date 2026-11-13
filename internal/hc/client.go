// Package hc is a minimal JSON-RPC 2.0 client for herdr-core, spoken over
// `hc connect` (plain mode: one JSON message per line on stdin/stdout).
package hc

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"sync"

	"github.com/shu-go/hccd/internal/sys"
)

type Client struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	reader *bufio.Reader
	mu     sync.Mutex
	nextID int
}

type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		Kind string `json:"kind"`
		Hint string `json:"hint"`
	} `json:"data"`
}

func (e *Error) Error() string {
	if e.Data.Hint != "" {
		return fmt.Sprintf("hc: %s (%s)", e.Message, e.Data.Hint)
	}
	return "hc: " + e.Message
}

type response struct {
	ID     *int            `json:"id"`
	Result json.RawMessage `json:"result"`
	Error  *Error          `json:"error"`
}

// Dial starts `hc connect` and performs the hello handshake.
func Dial() (*Client, error) {
	cmd := sys.Command("hc", "connect")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start hc: %w", err)
	}
	c := &Client{cmd: cmd, stdin: stdin, reader: bufio.NewReaderSize(stdout, 1<<20)}
	if err := c.Call("hello", map[string]any{"generation": 1}, nil); err != nil {
		c.Close()
		return nil, fmt.Errorf("hc hello: %w", err)
	}
	return c, nil
}

func (c *Client) Close() error {
	c.stdin.Close()
	return c.cmd.Wait()
}

// Call sends a request and waits for its response, skipping notifications.
func (c *Client) Call(method string, params, result any) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.nextID++
	id := c.nextID
	if params == nil {
		params = map[string]any{}
	}
	b, err := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": id, "method": method, "params": params})
	if err != nil {
		return err
	}
	if _, err := c.stdin.Write(append(b, '\n')); err != nil {
		return err
	}
	for {
		line, err := c.reader.ReadBytes('\n')
		if err != nil {
			return fmt.Errorf("hc %s: %w", method, err)
		}
		var r response
		if err := json.Unmarshal(line, &r); err != nil {
			return fmt.Errorf("hc %s: %w", method, err)
		}
		if r.ID == nil || *r.ID != id {
			continue
		}
		if r.Error != nil {
			return r.Error
		}
		if result != nil {
			return json.Unmarshal(r.Result, result)
		}
		return nil
	}
}

type Foreground struct {
	Name string   `json:"name"`
	Cwd  string   `json:"cwd"`
	Argv []string `json:"argv"`
	Pid  int      `json:"pid"`
}

type Terminal struct {
	Address    string            `json:"address"`
	State      string            `json:"state"`
	Cwd        string            `json:"cwd"`
	Pid        int               `json:"pid"`
	Argv       []string          `json:"argv"`
	Labels     map[string]string `json:"labels"`
	Foreground *Foreground       `json:"foreground"`
	Status     json.RawMessage   `json:"status"`
}

// Local reports whether the terminal is on this machine (not `machine/t7`).
func (t *Terminal) Local() bool { return !strings.Contains(t.Address, "/") }

// HasStatus reports whether a program in it reports a status (an agent);
// a plain shell has none.
func (t *Terminal) HasStatus() bool {
	s := strings.TrimSpace(string(t.Status))
	return s != "" && s != "null"
}

// CurrentDir is where the shell is now (foreground.cwd), else where it started.
func (t *Terminal) CurrentDir() string {
	if t.Foreground != nil && t.Foreground.Cwd != "" {
		return t.Foreground.Cwd
	}
	return t.Cwd
}

func (t *Terminal) Tab() string   { return t.Labels["herdr.tab"] }
func (t *Terminal) Space() string { return t.Labels["herdr.space"] }

func (c *Client) List() ([]Terminal, error) {
	var r struct {
		Terminals []Terminal `json:"terminals"`
	}
	if err := c.Call("terminal.list", nil, &r); err != nil {
		return nil, err
	}
	return r.Terminals, nil
}

func (c *Client) Get(addr string) (*Terminal, error) {
	var t Terminal
	if err := c.Call("terminal.get", map[string]any{"terminal": addr}, &t); err != nil {
		return nil, err
	}
	return &t, nil
}

// Send pastes text into the terminal and presses Enter.
func (c *Client) Send(addr, text string) error {
	return c.Call("terminal.send", map[string]any{"terminal": addr, "text": text}, nil)
}
