# Delve Command Reference

Curated reference of dlv commands grouped by purpose. See `dlv help` and `dlv help <cmd>` inside a session for canonical docs.

## Breakpoints

| Command                              | Notes                                                                 |
|--------------------------------------|-----------------------------------------------------------------------|
| `break <loc>`                        | Set breakpoint. `<loc>` = `function`, `file:line`, `*addr`, `pkg.Fn`. |
| `break <loc> if <cond>`              | Conditional breakpoint.                                               |
| `breakpoints` / `bp`                 | List breakpoints.                                                     |
| `clear <id>`                         | Delete one breakpoint.                                                |
| `clearall [<loc>]`                   | Delete all (or all at a location).                                    |
| `condition <id> <expr>`              | Set/replace condition on existing breakpoint.                         |
| `cond <id> -hitcount <op> <n>`       | Break on N-th hit, e.g. `-hitcount > 100`.                            |
| `on <id> <cmd>`                      | Run `<cmd>` whenever breakpoint id is hit.                            |
| `trace <loc>`                        | "Tracepoint" — log on hit without stopping.                           |
| `watch -r expr` / `-w` / `-rw`       | Read / write / read-write watchpoint (Linux/amd64 only).              |

Location formats:

- `main.main`, `(*Server).Handle` — function (receiver in parens for methods).
- `file.go:42` — line in file (basename ok if unambiguous).
- `github.com/org/repo/pkg/file.go:42` — fully qualified.
- `+10`, `-5` — N lines from current PC.

## Execution Control

| Command                  | Notes                                                |
|--------------------------|------------------------------------------------------|
| `continue` / `c`         | Resume until next breakpoint, panic, or exit.        |
| `next` / `n`             | Step to next source line in current frame.           |
| `step` / `s`             | Step into the called function.                       |
| `stepout` / `so`         | Run until the current function returns.              |
| `step-instruction` / `si` | One machine instruction (advanced).                 |
| `rev <cmd>`              | Reverse execution (`rev next`, `rev continue`) — requires recording mode (`dlv record` / `rr` backend). |
| `call <expr>`            | Call a function from the debugger (experimental; may corrupt state).|
| `restart` / `r`          | Restart the target (debug/test modes only).          |
| `restart -c <ckpt>`      | Restart at a checkpoint (rr backend).                |

## Inspection

| Command                           | Notes                                              |
|-----------------------------------|----------------------------------------------------|
| `print <expr>` / `p`              | Evaluate Go expression and print result.           |
| `print %x <expr>`                 | Format specifier: `%x`, `%o`, `%b`, `%c`, `%d`.    |
| `locals [-v]`                     | Local variables (`-v` shows unreferenced).         |
| `args [-v]`                       | Function arguments.                                |
| `whatis <expr>`                   | Type of expression.                                |
| `vars <regex>`                    | Package-level variables matching regex.            |
| `funcs <regex>`                   | Function names matching regex.                     |
| `types <regex>`                   | Type names matching regex.                         |
| `regs`                            | CPU registers.                                     |
| `display -a <expr>`               | Print `<expr>` after every step.                   |
| `display -d <id>`                 | Remove a display.                                  |

Expression notes:

- Full Go syntax is supported: indexing, slicing, field access, function calls (within limits), arithmetic.
- Built-ins: `len()`, `cap()`, `complex()`, `imag()`, `real()`.
- Cast with `(*pkg.Type)(addr)` for opaque pointers.

## Stack / Frames

| Command            | Notes                                            |
|--------------------|--------------------------------------------------|
| `stack [n]` / `bt` | Stack trace (optionally limit to n frames).      |
| `stack -full`      | With locals/args per frame.                      |
| `frame <n>`        | Switch to frame n (counted from 0 = current).    |
| `up [n]`           | Move up n frames.                                |
| `down [n]`         | Move down n frames.                              |
| `frame <n> <cmd>`  | Run `<cmd>` in the context of frame n.           |

## Goroutines / Threads

| Command                          | Notes                                                |
|----------------------------------|------------------------------------------------------|
| `goroutines` / `grs`             | List goroutines.                                     |
| `grs -t`                         | Include short stack traces.                          |
| `grs -with user`                 | Only goroutines started by user code.                |
| `grs -with running`              | Only goroutines currently running.                   |
| `grs -with status <s>`           | Filter by status (idle, runnable, running, waiting). |
| `goroutine [n]`                  | Show current goroutine, or switch to n.             |
| `goroutine n <cmd>`              | Run `<cmd>` in the context of goroutine n.           |
| `threads`                        | List OS threads.                                     |
| `thread <id>`                    | Switch to thread.                                    |

## Source Navigation

| Command               | Notes                                          |
|-----------------------|------------------------------------------------|
| `list` / `ls`         | Source around current PC.                      |
| `list <loc>`          | Source around a location.                      |
| `disassemble` / `disass` | Assembly for current function.              |
| `source <file>`       | Run dlv commands from a file (a "rc" script).  |

## Mutation

| Command               | Notes                                                  |
|-----------------------|--------------------------------------------------------|
| `set <var> = <expr>`  | Assign a new value to a variable in the running prog.  |
| `call <expr>`         | Call a function (see warning above).                   |

## Configuration

| Command                              | Notes                                                |
|--------------------------------------|------------------------------------------------------|
| `config max-string-len <n>`          | How many bytes of a string to print (default 64).    |
| `config max-array-values <n>`        | How many elements of a slice/array to print.         |
| `config max-variable-recurse <n>`    | How deep to recurse into nested structs.             |
| `config max-string-len 4096`         | Useful default before printing large messages.       |
| `config -list`                       | Show all config options.                             |
| `config substitute-path <from> <to>` | Remap source paths (vendored modules, build hosts).  |

## Session

| Command                  | Notes                                              |
|--------------------------|----------------------------------------------------|
| `help [<cmd>]`           | Help on a command.                                 |
| `sources <regex>`        | Source files known to the binary.                  |
| `libraries`              | Loaded shared libraries.                           |
| `dump <file>`            | Write a core dump.                                 |
| `checkpoint <name>`      | (rr backend) Mark a point for later return.        |
| `checkpoints`            | List checkpoints.                                  |
| `quit` / `q`             | Exit (will prompt to kill the process).            |
| `quit -c`                | Quit and clear breakpoints.                        |

## Patterns Worth Memorising

```
# Break on a method on a pointer receiver:
break (*Server).Handle

# Conditional on slice contents:
break loop.go:42 if items[i].ID == "abc"

# Print full struct with deep recursion:
config max-variable-recurse 5
print myStruct

# Find every goroutine waiting on the same channel:
grs -with status waiting

# Run a script of dlv commands at startup:
dlv debug . --init init.dlv
```
