# Getting Started

## Installation

### Pre-built binaries

Download the latest release for your platform from the
[releases page](https://github.com/kubeone/go-time-tracker/releases),
extract the archive, and place the `gtt` binary somewhere on your `PATH`.

```bash
# Linux amd64
curl -L https://github.com/kubeone/go-time-tracker/releases/latest/download/go-time-tracker_Linux_x86_64.tar.gz | tar xz
sudo mv gtt /usr/local/bin/
```

On Windows, download the `.zip` for your architecture, extract it, and place
`gtt.exe` in a directory that is on your `PATH` (e.g. `C:\Users\<you>\bin`).

### Install with Go

```bash
go install github.com/kubeone/go-time-tracker@latest
```

### Build from source

```bash
git clone https://github.com/kubeone/go-time-tracker.git
cd go-time-tracker
go build -o gtt .          # Linux / macOS
go build -o gtt.exe .      # Windows
```

## Requirements

| Platform | Requirement |
|----------|-------------|
| Linux | Any modern terminal emulator |
| macOS | Terminal.app or iTerm2 |
| Windows | [Windows Terminal](https://aka.ms/terminal) — legacy `cmd.exe` and the old PowerShell console do not support the ANSI codes used by the TUI |

## First run

```bash
gtt
```

On first launch, `gtt` creates a default configuration file (`config.yaml` in
your data directory) and four starter profiles: Development, Meetings, Code
Review, and Other.

## Data directory

| Platform | Path |
|----------|------|
| Linux / macOS | `~/.time-tracker/` |
| Windows | `%APPDATA%\gtt\` |

## Configuration

Edit `config.yaml` in your data directory to define your own profiles:

```yaml
profiles:
  - name: "Development"
    description: "Feature development and bug fixes"
  - name: "Meetings"
    description: "Team meetings and standups"
  - name: "Deep Work"
    description: "Focused, uninterrupted work sessions"
```

Changes are picked up the next time you start `gtt`.

## TUI keyboard shortcuts

| Key | Action |
|-----|--------|
| `↑` / `k` | Select previous profile |
| `↓` / `j` | Select next profile |
| `s` | Start timer for selected profile |
| `x` | Stop running timer |
| `r` | Resume last timer |
| `n` | Open / edit notes for selected profile |
| `e` | Browse and edit time entries |
| `←` / `h` | Go to previous day |
| `→` / `l` | Go to next day |
| `q` | Quit |

## Reports

```bash
gtt report              # today
gtt report yesterday    # yesterday
gtt report week         # current week (Mon–today)
gtt report 2026-06-20   # specific date
gtt report 2026-06-01:2026-06-07  # date range
```

## Storage format

Each day's data is stored as a Markdown file in the `days/` subfolder of your
data directory. The files are human-readable and can be edited by hand — `gtt`
reloads them automatically when it detects a change.

```markdown
# 2026-06-25 Thursday

## Development
> Feature development and bug fixes

- 09:00:00 – 10:30:00 (1h30m00s)
- 11:00:00 – 12:15:00 (1h15m00s)

**Total: 2h45m00s**

Worked on the new reporting feature.

## Meetings
> Team meetings and standups

- 10:30:00 – 11:00:00 (30m00s)

**Total: 30m00s**
```
