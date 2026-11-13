# hccd

English | [日本語](README_ja.md)

Open or switch to a Herdr tab by directory.

```
hccd [options] [<dir>|<file>]
```

- No argument: the current directory (when started from File Explorer, the folder shown in that Explorer window)
- `<dir>`: that directory
- `<file>`: opens the tab for its parent directory and runs the file in that tab's shell

If a tab whose shell is in that directory exists, hccd shows it; otherwise it creates a new tab. If no Herdr window is open, hccd starts one, and finally brings it to the front.

| Option | Description |
|---|---|
| `-s`, `--space` | Create a new space when no tab exists |
| `-n`, `--new` | Always create a new tab without looking for an existing one |
| `--name <name>` | Name of the new tab/space (default: folder name) |
| `-v`, `--verbose` | Explain what it does |
| `--version` | Print the version |
| `--register` / `--unregister` | (Windows) Add/remove "Open in Herdr" in the File Explorer context menu (can be combined with `-s`) |

## Requirements

- `hc` from [herdr-core](https://github.com/herdrdev/herdr-core) and `herdr-app` from [herdr-app](https://github.com/herdrdev/herdr-app) on your PATH

## Build

```powershell
go build -o bin\hccd.exe ./cmd/hccd                             # for terminals
go build -ldflags "-H=windowsgui" -o bin\hccdw.exe ./cmd/hccd   # for File Explorer (no console window)
```

On Linux / macOS: `go build -o bin/hccd ./cmd/hccd`.

## Using from File Explorer (Windows)

- Run `bin\hccd.exe --register` to add it to the context menu (if `hccdw.exe` is in the same folder, that one is registered). It appears in the menus for folders, folder backgrounds, drives and files. On Windows 11 it is under "Show more options".
- When `hccdw.exe` is started by double-click or from a shortcut, it uses the folder shown in the foreground Explorer window.
  If the foreground window is not Explorer (for example, a shortcut on the desktop), it uses the current directory (the shortcut's "Start in" folder; by default the folder of the exe).

## Limitations

- The directory is matched against the current location of Herdr's default shell (pwsh on Windows). If you start another shell such as cmd inside a tab and move around there, it cannot be detected.
- A shell is considered idle when no agent is in it and it has no child processes. While pwsh runs a built-in command (such as `Start-Sleep`), it is considered idle.
- Linux / macOS are not tested yet.
