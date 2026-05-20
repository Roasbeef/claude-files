---
name: go-debug
description: Interactively debug Go programs in a single context using Delve (dlv) driven through tmux. Use when a bug requires runtime inspection — stepping through code, examining variables, walking goroutines, attaching to a live process, or debugging a hanging integration test — rather than just reading the source. Triggers include "step through this", "set a breakpoint", "attach to the running server", "why is this goroutine stuck", "debug this failing test".
---

# Go Debug

This skill drives an interactive `dlv` debugging session through `tmux` from a single context, without delegating to a subagent. The agent calls one helper script (`scripts/dlv-session.sh`) per dlv command and gets back just the new output, so each step is small enough to reason about.

## When to Use

Reach for this skill when source reading alone is not enough:

- A bug only reproduces at runtime and the failure mode (panic, wrong value, hang) needs to be observed in flight.
- A test fails and the *why* is hidden — stepping into the failing line reveals state that logging would not.
- An integration test hangs, deadlocks, or behaves differently from the unit tests — attach to the process and inspect goroutines.
- A long-running server (lnd, eclair, a daemon under test) is misbehaving in a way that only shows up after warm-up. Use `attach` rather than restarting under the debugger.
- A concurrent bug needs goroutine-by-goroutine inspection (`goroutines`, `goroutine N`, `stack`).
- Stepping is wanted to confirm the *exact* control flow through a complex branch, not just "I think it takes this path."

Skip the skill when the bug is obvious from a `grep` or a unit test would be faster to write.

## Prerequisites

- `dlv` on PATH (`go install github.com/go-delve/delve/cmd/dlv@latest`).
- `tmux` on PATH.
- For `attach` on macOS: the target process must be debuggable by the current user (typically fine for processes the user started). On Linux, `ptrace` capability may need `sudo` or `kernel.yama.ptrace_scope=0`.

## The Helper Script

All session management goes through `~/.claude/skills/go-debug/scripts/dlv-session.sh`. It manages a single active session at a time, persists state in `$TMPDIR/claude-dlv/current`, strips ANSI noise from output, and waits for the next `(dlv)` prompt after each command so output is bounded.

```
dlv-session.sh start <dlv args>     # launch dlv, wait for first prompt
dlv-session.sh send  '<dlv command>'  # send one dlv command, return new output
dlv-session.sh send-raw '<text>'      # send keys without waiting for a prompt
dlv-session.sh output [N]             # last N lines of raw output (default 100)
dlv-session.sh status                 # is a session active?
dlv-session.sh stop                   # quit dlv, kill tmux session, clean state
```

`send` blocks until dlv prints the next `(dlv)` prompt or `DLV_SEND_TIMEOUT` (default 300s) elapses. Set `DLV_SEND_TIMEOUT=600` in the environment for unusually long `continue` operations.

## The Debugging Loop

A debugging session is a tight read–send–observe loop. After every `send`, read the returned output before deciding the next command — the program state may have moved in surprising ways.

1. **Start** a session in the mode matching the bug (see Launch Modes below).
2. **Set breakpoints** at the suspected fault site and at any choke points along the way.
3. **`continue`** to the first hit.
4. **Inspect** state: `print var`, `locals`, `args`, `stack`, `goroutines`.
5. **Step** in (`step`) or over (`next`) one line at a time when state diverges from expectation.
6. **Form a hypothesis** about what the program is doing wrong. Confirm it by inspecting the specific variable, slice, map key, or goroutine that the hypothesis implicates.
7. **Stop** the session before exiting, and only then edit code.

Edit-then-debug, not edit-while-debugging: source changes are not reflected in the currently-running binary. After editing, `stop` and `start` again.

## Launch Modes

### `debug` — debug a main package

Build and run the program under the debugger. Use for reproducible crashes or wrong-output bugs.

```
dlv-session.sh start debug .
dlv-session.sh start debug ./cmd/myapp
dlv-session.sh start debug ./cmd/myapp -- --flag=value --other arg
```

Arguments after `--` are forwarded to the program.

### `test` — debug a test

Build and run a `go test` binary under the debugger. The most common entry point — most bugs have or can quickly get a failing test.

```
dlv-session.sh start test ./pkg/foo
dlv-session.sh start test ./pkg/foo -- -test.run TestSpecificCase
dlv-session.sh start test ./pkg/foo -- -test.run TestX -test.v
```

For a single test, always narrow with `-test.run` — otherwise breakpoints fire on the first matching frame from any test.

### `attach` — attach to a running process

The most useful mode for integration tests and live services. Use when:

- An integration test launches a daemon (lnd, eclair, a microservice) as a child process and the daemon misbehaves. Find the PID (`pgrep`, `ps`, the test's log output) and attach.
- A long-running server is stuck or behaving incorrectly in a way that does not reproduce from a cold start.
- A bug only appears after specific runtime state has accumulated (cache warm-up, connections established, channels opened).

```
dlv-session.sh start attach <pid>
dlv-session.sh start attach <pid> ./path/to/binary
```

Passing the binary path is optional but speeds up symbol resolution and is required if dlv cannot find debug info from the running executable.

The target process is **paused** the moment dlv attaches. Set breakpoints first, then `continue` to release it. On `stop`, dlv detaches and the process resumes; if the process was started by a test, killing it via dlv's quit-prompt would tear down the test, so prefer `detach` semantics by quitting cleanly.

For attach to work, the binary must be built with debug info — `go build` keeps it by default, but `-ldflags="-s -w"` or `go install` with stripping removes it. If symbols are missing, rebuild the target without strip flags.

### `test` + `attach` workflow for integration tests

When an integration test launches a sub-binary that misbehaves:

1. Add a `time.Sleep` or breakpoint-friendly pause to the test just after the sub-binary starts (or use the test's existing log line that prints the PID).
2. Run the test normally: `go test ./integration -run TestThatHangs -v`.
3. In a separate context, find the sub-binary's PID: `pgrep -f my-daemon`.
4. `dlv-session.sh start attach <pid>` and set breakpoints in the daemon's code.
5. Let the test proceed (remove the sleep or hit `continue`).

### `connect` — connect to a `dlv` headless server

Useful when something else (CI runner, a Make target, an editor) started `dlv --headless --listen=:2345`. The target is already running and possibly remote.

```
dlv-session.sh start connect 127.0.0.1:2345
```

### `core` — post-mortem on a core dump

```
dlv-session.sh start core ./binary ./core
```

## Essential dlv Commands

A reference cheatsheet — see `references/dlv-commands.md` for the full list. Names that appear in the loop most often:

| Command                          | Purpose                                       |
|----------------------------------|-----------------------------------------------|
| `break main.foo`                 | Break on function `main.foo`                  |
| `break file.go:42`               | Break on file:line                            |
| `break file.go:42 if x > 10`     | Conditional breakpoint                        |
| `breakpoints`                    | List breakpoints                              |
| `clear 1`                        | Delete breakpoint by id                       |
| `clearall`                       | Delete all breakpoints                        |
| `continue` / `c`                 | Run until next breakpoint or exit             |
| `next` / `n`                     | Step over (one source line, stay in frame)    |
| `step` / `s`                     | Step into                                     |
| `stepout` / `so`                 | Step out of current function                  |
| `restart` / `r`                  | Restart the program (modes: debug/test)       |
| `print expr` / `p expr`          | Evaluate and print expression                 |
| `locals`                         | All locals in current frame                   |
| `args`                           | Function arguments                            |
| `whatis x`                       | Type of expression                            |
| `list` / `ls`                    | Source around current PC                      |
| `stack` / `bt`                   | Stack trace                                   |
| `frame N`                        | Switch to frame N in the stack                |
| `goroutines` / `grs`             | List all goroutines                           |
| `goroutine N`                    | Switch to goroutine N                         |
| `set x = 42`                     | Mutate a variable                             |
| `condition 1 x > 5`              | Add a condition to breakpoint 1               |
| `trace pkg.Func`                 | Print every call to Func without stopping     |

## Worked Patterns

### Wrong-output bug

```bash
dlv-session.sh start test ./pkg -- -test.run TestFindMax
dlv-session.sh send 'break pkg.findMax'
dlv-session.sh send 'continue'
dlv-session.sh send 'args'         # confirm input
dlv-session.sh send 'next'
dlv-session.sh send 'print max'    # watch the value evolve
dlv-session.sh send 'next'
# ... iterate. When the wrong value appears, stop.
dlv-session.sh stop
```

### Nil pointer panic

The panic location is in the stack trace; the *cause* is upstream. Set a breakpoint a few lines earlier, then step.

```bash
dlv-session.sh start test ./pkg -- -test.run TestPanics
dlv-session.sh send 'break pkg/file.go:LINE_BEFORE_PANIC'
dlv-session.sh send 'continue'
dlv-session.sh send 'print suspectedNilVar'
dlv-session.sh send 'whatis suspectedNilVar'
dlv-session.sh send 'next'
```

### Goroutine deadlock

```bash
dlv-session.sh start attach <pid>
dlv-session.sh send 'goroutines'           # find blocked goroutines
dlv-session.sh send 'goroutine 12'         # switch
dlv-session.sh send 'stack'                # where is it blocked?
dlv-session.sh send 'goroutine 17'
dlv-session.sh send 'stack'                # who's holding the lock it wants?
```

The pair `goroutines -t` (with traces) plus `bt` on each suspect usually fingers the cycle.

### Slice / map inspection

```bash
dlv-session.sh send 'print len(items)'
dlv-session.sh send 'print cap(items)'
dlv-session.sh send 'print items[0:5]'
dlv-session.sh send 'print byId[key]'
```

For very large containers, `print` may truncate. Use `config max-string-len 4096` / `config max-array-values 200` before the `print`.

### Conditional breakpoint to skip warm-up

```bash
dlv-session.sh send 'break server.go:120'
dlv-session.sh send 'condition 1 req.Method == "POST" && len(req.Body) > 0'
dlv-session.sh send 'continue'
```

### Catching a specific iteration

```bash
dlv-session.sh send 'break loop.go:30 if i == 4242'
dlv-session.sh send 'continue'
```

## Common Pitfalls

- **Stale state file.** If a previous session crashed, the next `start` will refuse. Run `stop` (it tolerates missing tmux sessions) or `rm $TMPDIR/claude-dlv/current`.
- **Source not visible.** Breakpoints at file paths must use the path as the compiler sees it. If `break foo.go:30` fails, try the fully-qualified package path (`break github.com/org/repo/pkg/foo.go:30`) or set on a function (`break pkg.Foo`).
- **No symbols when attaching.** A binary built with `-ldflags="-s -w"` cannot be debugged. Rebuild without stripping.
- **`continue` never returns.** The program is running but no breakpoint fires — either the path is not taken, or the breakpoint location resolved to nothing (check `breakpoints`). Send Ctrl-C via `send-raw $'\x03'` to interrupt, then re-set the breakpoint.
- **Editing source mid-session.** Changes do not take effect until the next `start`. After every code edit, `stop` and start over.
- **macOS code signing.** First-time `dlv` use may require approving the `debugserver` codesigning. If `start` fails with a security error, run `dlv version` in a regular terminal once to surface the macOS prompt.
- **Hung tests in CI under attach.** If a test framework has a global timeout, attaching for too long will trip it. Lengthen the timeout (`-test.timeout=0` for go test) before reproducing.

## Cleanup

Always end a session with `dlv-session.sh stop`. It quits dlv, answers the kill-process prompt, and tears down the tmux session and state file. Leaving a session running consumes a tmux server slot and keeps an attached process paused.

If a `start` ever hangs, the tmux session and output file under `$TMPDIR/claude-dlv/` are visible — list active tmux sessions with `tmux ls` and kill stragglers with `tmux kill-session -t <name>`.
