# Field Kit

**A set of everyday-carry tools for the terminal.** One binary, tmux integration, plain markdown files.

Field Kit gives you five small tools — todo list, field notes, junk drawer, scratch pad, calculator — that keep your ad hoc work organized without requiring you to think about where things go. Everything uses a staging/flushing model: you write into a scratchpad, and the system periodically files it away.

---

## Architecture

### Two repos

- **`tmux-fieldkit`** (public) — The Go source code. Compiles to a single static binary called `kit`.
- **`~/kit`** (private, per-user) — Your data. Plain markdown files in a git repo. Created by `kit init`.

### Dependencies

Assumed to be already installed:

- `tmux`
- `git`

### Tech stack

- **Go** for the binary
- **Bubble Tea** (charmbracelet) for the calculator TUI
- **Lip Gloss** (charmbracelet) for styling
- No shell scripts. All logic lives in the Go binary.

---

## Data directory layout

```
~/kit/
  .git/
  config/             # configuration files
    kit.toml          # kit config
    keybindings.tmux  # tmux config snippet for manual sourcing
  todo/
    stage.md          # current staging list
    list.md           # combined todo list
  notes/
    today/            # staging notes, one file per session
    topics/           # accumulated topic files
  junk/               # free-form file layout
  scratch/            # ignored by git
    stage/            # current scratch files, automatically trashed after 30 days
    trash/            # old scratch files, deleted after 60 days
  calc/
    stage.md          # current calculator session
    history/          # monthly archives
      YYYY/           # automatically deleted after 180 days
        MM.md
```

---

## Go project layout

```
tmux-fieldkit/
  cmd/
    kit/
      main.go           # CLI entry point
  internal/
    todo.go             # todo open/flush logic
    scratch.go          # scratch pad open/flush logic
    junk.go             # junk drawer open/flush logic
    notes.go            # field notes open/flush logic
    calc/
      repl.go           # Bubble Tea TUI
      eval.go           # expression evaluator
    gitutil.go          # auto-commit logic
    tmux.go             # tmux popup/session/keybinding helpers
    config.go           # kit.toml read/write, setup wizard
    flush.go            # shared flush utilities (timer checks, etc.)
  go.mod
  go.sum
  README.md
  LICENSE
```

---

## General CLI commands

```
kit init              # create ~/kit, git init, scaffold directories
kit setup             # interactive setup wizard — asks about tmux prefix,
                      #   editor, timezone; writes kit.toml; patches tmux.conf
kit flush <tool>      # flush a specific tool
kit flush all         # flush all tools
kit status            # show per-tool details about what hasn't been flushed yet
                      #   todo: line count in stage.md
                      #   notes: count and names of files in today/
                      #   calc: line count in stage.md
                      #   scratch,junk: n/a (no staging)
kit install           # alias for init + setup
kit uninstall         # remove tmux config lines, optionally delete ~/kit
```

## Tool-specific CLI commands

```
# todo list
kit toggle todo       # open todo popup if closed, close if open
kit cycle todo        # flush stage → list, open fresh stage

# field notes
kit toggle notes      # open today's notes if closed, close if open
                      # create a new file for today's notes if none exist
kit new note          # create a new file for today's notes, open it

# junk drawer
kit toggle junk       # switch to junk drawer session, creating it if necessary,
                      # or hide it if it's already focused
kit new junk          # kill junk drawer session if it exists, create a new one

# scratch pad
kit toggle scratch    # switch to scratch pad session, creating it if necessary,
                      # or hide it if it's already focused
kit new scratch       # kill scratch pad session if it exists, create a new one

# calculator
kit toggle calc       # switch to calc session, creating it if necessary,
                      # or hide it if it's already focused
kit new calc          # kill calc session if it exists, create a new one
```

## Example tmux keybindings

```tmux
bind -n M-g run-shell "kit toggle todo"
bind -n M-G run-shell "kit cycle todo"
bind -n M-f run-shell "kit toggle notes"
bind -n M-F run-shell "kit new note"
bind -n M-d run-shell "kit toggle junk"
bind -n M-D run-shell "kit new junk"
bind -n M-s run-shell "kit toggle scratch"
bind -n M-S run-shell "kit new scratch"
bind -n M-a run-shell "kit toggle calc"
bind -n M-a run-shell "kit new calc"
```

`kit new scratch` may prompt for a file extension or type (e.g. "code", "text", "markdown") and name the file accordingly (`scratch-2024-06-30-001-code.py` vs `scratch-2024-06-30-001-text.txt`).

---

## Git auto-commit

A shared utility called on every tool open, close, and flush.

### Logic

```
function maybe_commit():
  if no changes (git status --porcelain is empty):
    return
  if this is a flush:
    commit("flush: <tool>")
  else if last commit was > 1 hour ago:
    commit("auto: <tool>")
  else:
    return
```

### Commit messages

Short, predictable, `git log --oneline` friendly:

- `flush: todo`
- `flush: notes`
- `flush: calc`
- `auto: todo` (time-based checkpoint)
- `auto: notes`

All files are staged (`git add -A`) before each commit.

---

## Tool: Todo

### Files

- `~/kit/todo/stage.md` — current staging list
- `~/kit/todo/list.md` — combined todo list

### Behavior

- Always opens in a **tmux popup**.
- `kit toggle todo` — if already open, close. if `stage.md` is non-empty, open it. Otherwise open `list.md`.
- `kit cycle todo` — flush `stage.md` into `list.md`, then open a fresh `stage.md`.

### Auto-flush

When opening a todo popup, check the mtime of `stage.md`. If it's older than **2 hours**, flush automatically before opening.

---

## Tool: Field Notes

### Files

- `~/kit/notes/today/YYYY-MM-DD-HHMMSS.md` — staging notes (one file per session)
- `~/kit/notes/topics/<topic>.md` — accumulated notes by topic

### Behavior

- Always opens in a **fullscreen tmux session** named `field notes`.
- `kit toggle notes` — hide the `field notes` session if it's visible. Otherwsie, open the most recent file in `today/` for the current date, or create a new one if none exists.
- `kit new note` — create a new timestamped file in `today/` and open it, even if another session is already open.
- `kit flush notes` — process all files in `today/`:
  1. Parse each file for markdown `# Heading` lines.
  2. Normalize each heading to a slug (lowercase, hyphens, strip special chars) → this is the topic name.
  3. Content before the first heading gets topic name `notes` (i.e., implicit `# Notes`).
  4. Append each section to `~/kit/notes/topics/<topic>.md`, with a date header. Topic files are created if they don't exist. Heading collisions are expected and fine — content is just appended.
  5. Delete (or move/archive) the processed files from `today/`.

### Auto-flush

When opening the field notes session, check if there are files in `today/` from a **previous day**. If so, flush them automatically. The canonical flush time is 4 AM local (as configured in `kit.toml`), but the check is triggered on open, not by a cron job.

### Topic management

- **New topics** are created automatically when a heading doesn't match an existing topic file.
- **Merging topics** is manual — just edit the files, combine content, delete the old file.
- **Splitting topics** is manual — edit a topic file, cut sections into new files.
- Git history serves as the log/audit trail, so there's no separate `logs/` directory.

---

## Tool: Junk Drawer

### Files

- `~/kit/junk/` — freeform files, any extension, any structure. No staging, no flushing. Just a place to put things.

### Behavior

- `kit toggle junk` — Hide the junk drawer if it's currently focused. Otherwise, switch to the `junk` tmux session, creating it if necessary. The session starts in the `~/kit/junk/` directory, so the user's editor determines how files are displayed.
- `kit new junk` — Kill the `junk` tmux session if it exists, then create a new one and switch to it.


### No flushing

The junk drawer has no staging/flushing cycle. It's just a place to put things.

---

## Tool: Scratch Pad

### Files

- `~/kit/scratch/stage/YYYY-MM-DD-HHMMSS` — current scratch files, one per session, with timestamped names, automatically trashed after 30 days
- `~/kit/scratch/trash/YYYY-MM-DD-HHMMSS` — trashed scratch files, automatically deleted after 60 days

---

## Tool: Calculator

### Files

- `~/kit/calc/stage.md` — current calculator session
- `~/kit/calc/history/YYYY/MM.md` — monthly archives

### Behavior

- Always opens in a **tmux popup**.
- `kit open calc` — launch the calculator REPL TUI, backed by `stage.md`.
- `kit flush calc` — file the contents of `stage.md` into the appropriate monthly history file(s), then clear `stage.md`.

### The calculator REPL

A Bubble Tea TUI that functions as a desk calculator with memory. On open, it reads `stage.md`, strips any `= result` lines, re-evaluates all expressions top to bottom, and rewrites the results. Expressions are the source of truth; results are always derived. Manual edits to expression lines take effect on next open.

**Supported features:**

- Arithmetic operators: `+`, `-`, `*`, `/`, `%`
- Parentheses for grouping
- Variable assignment: `x = 42`
- Variable reference: `x * 2` → `84`
- Comments: lines starting with `#`
- Blank lines (preserved for visual spacing)
- Date-change comments: when the date changes during a session, a `# --- YYYY-MM-DD ---` comment is inserted automatically.
- Units of measure for bytes and time, with derived rate units and explicit conversion via `in`:
  - Bytes: `b`, `B`, `KB`, `MB`, `GB`, `TB` (SI, base-10) and `KiB`, `MiB`, `GiB`, `TiB` (binary, base-2)
  - Time: `ms`, `s`, `min`, `hr`, `day`
  - Rates: `KB/s`, `MB/s`, `GB/s`, `Kbps`, `Mbps`, `Gbps`
  - Conversion: `expr in <unit>` — e.g. `file / speed in min`
  - Incompatible unit operations (e.g. adding bytes to seconds) are an error

**Not supported (intentionally):**

- Functions
- Strings
- Conditionals
- Loops
- Imports
- Unit families outside bytes and time (no length, weight, currency, etc.)

**Interaction model:**

- Line-at-a-time, append-only. Type an expression, hit enter, `= result` appears below.
- Up/down arrow keys cycle through input history (previous expressions).
- A recalled line can be edited before submitting.
- `ctrl+r` — reverse history search (stretch goal).
- `/` at the start of a line — inline history search, same behavior (stretch goal).

**Display:**

Each expression line shows the result on the next line. The file on disk is a readable markdown document:

```markdown
# 2026-04-02

x = 100
x * 1.08
= 108

y = x * 12
= 1200

y / 52
= 23.076923...

# quick estimate
y * 0.3
= 360

# transfer time
file = 100 GB
speed = 1 Gbps
file / speed
= 800 s

file / speed in min
= 13.333 min
```

### Flushing

When flushing, the contents of `stage.md` are split by the date-change comments and appended to the appropriate `history/YYYY/MM.md` file. Then `stage.md` is cleared.

---

## Config: `kit.toml`

TOML format. All fields have sensible defaults; the file is optional.

```toml
# Editor command (default: nvim)
editor = "nvim"

# Timezone for auto-flush time checks (default: system timezone)
timezone = "America/Chicago"

# Auto-flush time for field notes (default: "04:00")
notes_flush_time = "04:00"

# Todo auto-flush timeout in minutes (default: 120)
todo_flush_timeout = 120

# Auto-commit interval in minutes (default: 60)
commit_interval = 60

# Tmux session used for popup tool windows (default: __kit__)
popup_session = "__kit__"
```

---

## Install / setup flow

```bash
# Option A: go install
go install github.com/<user>/tmux-fieldkit/cmd/kit@latest

# Option B: clone and build
git clone https://github.com/<user>/field-kit.git
cd field-kit
go build -o kit ./cmd/kit
mv kit ~/.local/bin/  # or wherever

# Initialize data directory
kit init

# Interactive setup (configures tmux, writes .kitrc)
kit setup
```

`kit setup` does the following:

1. Asks for preferred editor (default: nvim).
2. Asks for timezone (default: detected from system).
3. Asks for data directory (default: ~/kit).
4. Writes `kit.toml` to the data directory.
5. Generates tmux keybinding config.
6. Offers to add a `source-file` line to `~/.tmux.conf` (or prints it for manual addition).

`kit uninstall` removes the `source-file` line from `.tmux.conf` and optionally deletes the data directory.


## Stretch goals / future features

- `kit sync` command to push/pull the `~/kit` git repo, with conflict handling.
- Calendar tool.


## Open questions

### How does the `kit` binary know where the data directory is?

Options:

  - `@fieldkit_home` tmux global variable set in the tmux config snippet (but this doesn't help for non-tmux commands like `kit flush todo`)
  - Always `~/kit`
  - Look for a `KIT_HOME` env var, defaulting to `~/kit`
  - Require an explicit `--data-dir` flag on every command (hide behind aliases and keybindings)
  - Move the config file to `~/.kitrc` or `~/.config/kit/config.toml` and read the data directory from there.
