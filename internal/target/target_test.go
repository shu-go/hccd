package target

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNormalize(t *testing.T) {
	cases := []struct {
		a, b, goos string
		same       bool
	}{
		{`C:\Users\Someone\`, `c:\users\someone`, "windows", true},
		{`C:\`, `C:`, "windows", true},
		{`C:\a b\日本語`, `c:\A B\日本語\`, "windows", true},
		{`C:\a`, `C:\ab`, "windows", false},
		{"/home/u/src/", "/home/u/src", "linux", true},
		{"/home/u/Src", "/home/u/src", "linux", false},
		{"/Users/u/Src", "/users/u/src", "darwin", true},
		{"/", "/", "linux", true},
	}
	for _, c := range cases {
		if c.goos == "windows" && filepath.Separator != '\\' {
			continue // filepath.Clean is OS-specific
		}
		if c.goos != "windows" && filepath.Separator == '\\' {
			continue
		}
		got := normalize(c.a, c.goos) == normalize(c.b, c.goos)
		if got != c.same {
			t.Errorf("%s: %q vs %q = %v, want %v", c.goos, c.a, c.b, got, c.same)
		}
	}
}

func TestRunCommand(t *testing.T) {
	cases := []struct{ shell, file, want string }{
		{"pwsh.exe", `C:\a b\it's.ps1`, `& 'C:\a b\it''s.ps1'`},
		{`C:\Program Files\PowerShell\7\pwsh.exe`, `C:\x.bat`, `& 'C:\x.bat'`},
		{"powershell.exe", `C:\x.ps1`, `& 'C:\x.ps1'`},
		{"cmd.exe", `C:\a b\x.bat`, `"C:\a b\x.bat"`},
		{"bash", "/tmp/it's.sh", `'/tmp/it'\''s.sh'`},
		{"-zsh", "/tmp/x.sh", `'/tmp/x.sh'`},
		{"fish", `/tmp/it's.sh`, `'/tmp/it\'s.sh'`},
		{"", "/tmp/x.sh", `'/tmp/x.sh'`},
	}
	for _, c := range cases {
		if got := RunCommand(c.shell, c.file); got != c.want {
			t.Errorf("RunCommand(%q, %q) = %s, want %s", c.shell, c.file, got, c.want)
		}
	}
}

func TestResolve(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "sub dir")
	os.Mkdir(sub, 0o755)
	file := filepath.Join(sub, "x.ps1")
	os.WriteFile(file, nil, 0o644)

	if tg, err := Resolve("", sub); err != nil || tg.Dir != sub || tg.File != "" {
		t.Errorf("empty arg: %+v %v", tg, err)
	}
	if tg, err := Resolve("sub dir", dir); err != nil || tg.Dir != sub {
		t.Errorf("relative dir: %+v %v", tg, err)
	}
	if tg, err := Resolve(file, dir); err != nil || tg.Dir != sub || tg.File != file {
		t.Errorf("file: %+v %v", tg, err)
	}
	if _, err := Resolve(filepath.Join(dir, "missing"), dir); err == nil {
		t.Error("missing: want error")
	}
}
