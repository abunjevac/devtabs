# devtabs

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/Go-1.26+-00ADD8?logo=go&logoColor=white)](go.mod)
[![Platform](https://img.shields.io/badge/platform-Linux-lightgrey?logo=linux&logoColor=white)](https://github.com/abunjevac/devtabs)
[![Release](https://img.shields.io/github/v/release/abunjevac/devtabs)](https://github.com/abunjevac/devtabs/releases)
[![Build](https://github.com/abunjevac/devtabs/actions/workflows/build.yml/badge.svg)](https://github.com/abunjevac/devtabs/actions/workflows/build.yml)

A Linux desktop terminal launcher that opens multiple named terminal tabs from a single YAML config file. Each tab runs
its own shell with an optional command, working directory, and startup behaviour.

Built with GTK4 and VTE.

---

## Features

- **Named tabs** — each tab has a name shown as a label
- **State indicator dot** — grey (idle), green (running), yellow (startup delay), red (dead)
- **Run / Stop** — execute or interrupt the configured command per tab, or all tabs at once
- **Auto-run on startup** — optional `run_on_startup` with configurable `startup_delay`
- **Font scaling** — `Ctrl++` / `Ctrl+-` to adjust terminal font size at runtime
- **Startup tab** — configure which tab receives focus on launch
- **Profile filtering** — tag tabs with profiles and pass `--profile work,dev` to show only matching tabs
- **Hamburger menu** — open a terminal or file manager in the config file's directory, or restart
- **Keyboard shortcuts** — `Alt+1`–`9` to switch tabs, `Alt+R/S/A/X` for run/stop, `Ctrl+Q` to quit
- **Version in titlebar** — shows the release version at a glance

---

## Requirements

- Linux
- GTK4 (`libgtk-4-dev`)
- VTE GTK4 (`libvte-2.91-gtk4-dev`)
- Go 1.26+
- [Task](https://taskfile.dev) (optional, for the `task` commands)

Install system dependencies on Ubuntu/Debian:

```bash
sudo apt install libgtk-4-dev libvte-2.91-gtk4-dev
```

---

## Installation

### Download a release

Grab the latest Linux binary from the [Releases](../../releases) page and put it on your `$PATH`:

```bash
chmod +x devtabs
sudo mv devtabs /usr/local/bin/
```

### Build from source

```bash
git clone https://github.com/abunjevac/devtabs.git
cd devtabs
task build          # requires Task — https://taskfile.dev
# or: go build -o devtabs ./cmd/devtabs
```

---

## Usage

```
devtabs [--config <path>] [--root <dir>] [--profile <profiles>]
```

| Flag              | Default               | Description                                                                                                     |
|-------------------|-----------------------|-----------------------------------------------------------------------------------------------------------------|
| `--config`, `-c`  | `./devtabs.yaml`      | Path to the YAML config file                                                                                    |
| `--root`, `-r`    | config file directory | Base directory for resolving relative `working_dir` paths                                                       |
| `--profile`, `-p` | _(all tabs)_          | Comma-separated profile names; only matching tabs are shown (tabs with no `profiles` field are always included) |

```bash
devtabs                                    # uses ./devtabs.yaml, shows all tabs
devtabs --config ~/projects/work.yaml
devtabs --profile work                     # show tabs tagged "work" plus untagged tabs
devtabs --profile work,dev                 # union of "work" and "dev" profiles
devtabs --version
```

---

## Configuration

Create a `devtabs.yaml` file. Only `tabs` is required; everything else is optional.

```yaml
startup_tab: server        # focus this tab on launch (must match a tab name)
font: "Monospace"          # terminal font family (default: Monospace)
font_size: 13              # points (default: 12)
window_width: 1400         # pixels (default: 1200)
window_height: 900         # pixels (default: 800)

tabs:
  - name: server
    command: npm run dev
    working_dir: ~/projects/myapp   # ~ is expanded to your home directory
    run_on_startup: true
    startup_delay: 0s
    shell: /bin/zsh
    shell_args: [ "-l" ]
    profiles: [ work ]              # only shown when --profile work (or no --profile)

  - name: worker
    command: npm run worker
    working_dir: ~/projects/myapp
    run_on_startup: true
    startup_delay: 3s               # wait 3 s after launch before running
    shell: /bin/zsh
    shell_args: [ "-l" ]
    profiles: [ work ]

  - name: shell
    command: ls -la
    working_dir: ~/projects/myapp
    run_on_startup: false           # idle on start; press Run or Alt+R to execute
    shell: /bin/zsh
    shell_args: [ "-l" ]
    # no profiles: always shown regardless of --profile
```

### Tab fields

| Field            | Required | Default               | Description                                                                   |
|------------------|----------|-----------------------|-------------------------------------------------------------------------------|
| `name`           | yes      | —                     | Unique label shown on the tab                                                 |
| `command`        | yes      | —                     | Command written to the shell when Run is triggered                            |
| `working_dir`    | no       | config file directory | Shell's working directory                                                     |
| `run_on_startup` | no       | `false`               | Run the command automatically when the app starts                             |
| `startup_delay`  | no       | `0s`                  | Delay before auto-running (Go duration: `500ms`, `2s`, `1m`)                  |
| `shell`          | no       | `/bin/zsh`            | Shell binary                                                                  |
| `shell_args`     | no       | `["-l"]`              | Arguments passed to the shell                                                 |
| `profiles`       | no       | _(always shown)_      | List of profile names; tab is shown only when `--profile` matches one of them |

`run_on_startup` accepts `true`/`false`, `yes`/`no`, `on`/`off`, or `1`/`0`.

---

## Keyboard shortcuts

| Shortcut          | Action                     |
|-------------------|----------------------------|
| `Alt+1` – `Alt+9` | Switch to tab N            |
| `Alt+R`           | Run command in current tab |
| `Alt+S`           | Stop (Ctrl+C) current tab  |
| `Alt+A`           | Run all idle tabs          |
| `Alt+X`           | Stop all running tabs      |
| `Ctrl++`          | Increase font size         |
| `Ctrl+-`          | Decrease font size         |
| `Ctrl+Q`          | Quit                       |

## Toolbar menu

The `☰` button on the right side of the toolbar opens a menu with:

| Action             | Description                                                         |
|--------------------|---------------------------------------------------------------------|
| Open Terminal Here | Opens a new terminal in the directory containing the config file    |
| Open Files Here    | Opens the file manager in the directory containing the config file  |
| Restart            | Closes all terminals and relaunches devtabs with the same arguments |

Supported terminal emulators (tried in order): `x-terminal-emulator`, `gnome-terminal`, `xfce4-terminal`, `konsole`.

---

## Releasing

Tag a commit to trigger an automated GitHub Actions build:

```bash
git tag v1.2.3
git push origin v1.2.3
```

The workflow builds a Linux binary, creates a GitHub release with auto-generated notes, and prunes releases beyond the
three most recent.

---

## Development

```bash
task build    # build binary
task run      # build and run with ./devtabs.yaml
task test     # run tests with race detector
task lint     # golangci-lint + go vet
task check    # lint + test + osv-scanner
task tidy     # go mod tidy
```

---

## License

[MIT](LICENSE) © Alan Bunjevac
