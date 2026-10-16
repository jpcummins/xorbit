# Project Rules

## How to use this file

This file is authoritative for repository work. Read it in full before changing code.

- The key words **MUST**, **MUST NOT**, **SHOULD**, **SHOULD NOT**, and **MAY** are used as defined in RFC 2119. A MUST rule has no exceptions unless the user explicitly grants one in the current conversation.
- **Precedence:** Data Containment overrides all other rules; Security takes priority over implementation preferences. When requirements conflict, follow the stricter rule.
- **When a request conflicts with a rule:** stop, name the rule, and ask the user how to proceed. You MUST NOT silently weaken, work around, or reinterpret a rule to complete a task.
- **Before finishing any code change**, run the verification commands in [Verification](#verification) and walk the [Review checklist](#review-checklist).
- **Scope:** these rules describe the program and its implementation. Ask before introducing behavior or architecture not covered here.

## Data Containment Principle

This program mixes entropy from multiple semi-trusted sources and XORs it together.

**No data generated or handled by this program (inputs, intermediate values, results, logs, errors, stack traces, timing) may leave the process through any channel another program could observe, or be placed anywhere the operating system could persist it.**

- The final entropy and mnemonic code are the most sensitive data the program holds. They MUST leave the process only through the controlling terminal (TTY): the source contributions and final entropy MAY be shown together on the XOR screen after both sources succeed, including redraws while that screen is active, and the mnemonic code MAY be shown once on the reveal screen after the terminal security check. They MUST NOT be written to stdout, stderr, a file, a pipe, or any other file descriptor, and MUST NOT be returned to a caller. See [The TTY interface](#the-tty-interface).
- Values derived from operator input MAY appear on the TTY only as allowed in [The TTY interface](#the-tty-interface).
- The process exit code is the only other signal the program emits. It MUST carry no data beyond success or a fixed failure category.
- Observable channels include, at minimum: stdout, stderr, files (including temp files and `tmpfs`), pipes, sockets, the network, environment variables, command-line arguments, signals that trigger runtime dumps, core dumps, swap, `/proc/<pid>/*`, ptrace, child processes, and terminal scrollback. The TTY is the single sanctioned exception to this list.
- "Timing" here means timing at the cryptographic layer, where secret-dependent timing leaks data. Process-level timing (wall-clock duration, time to exit) is an accepted residual risk.

## Logging

- Logging is forbidden. Do not import `log`, `log/slog`, or `log/syslog`, or implement a logger or log buffer. Use only fixed failure messages where the TTY rules permit them and fixed exit categories.

## Go Package Layout

Use a domain root, an application package, boundary adapters, and a composition root:

1. The root package holds shared domain types and interfaces.
2. `internal/app` owns application flow and the interfaces it consumes.
3. Adapter subpackages are grouped by external boundary; `main` composes them.
4. A shared `mock` subpackage holds hand-written fakes.

### Root package: domain only

- The root package contains shared domain types and interfaces only. It MUST NOT import other module packages or perform I/O; code that touches the OS, filesystem, clock, or another external boundary belongs in an adapter.
- Put an interface in the package that owns its abstraction: shared domain interfaces in the root, application ports in `internal/app`.

### Application package: `internal/app`

- `internal/app` owns the fixed application flow, screens, input, workspace, validation, and the ports it consumes. It MAY import root-domain types and allowed standard-library packages, but MUST NOT import concrete adapters.
- Keep flow and views in this package, split into files for readability; do not add a generic presentation framework.
- Use non-stuttering names (for example, avoid `app.App`).

### Subpackages: grouped by dependency

- Group adapters by external boundary, and isolate OS-facing packages such as `os` and `syscall`. Each adapter MUST implement a consumer-owned port or shared domain interface, MAY import the root, `internal/app`, and its boundary dependency, and MUST NOT depend on another adapter.
- Adapters are limited to the standard-library boundaries allowed by [Security](#security); network and database adapters are forbidden. Process hardening lives in its own package and runs first from `main`.
- Avoid generic role- or feature-named packages and stuttering names.

### Shared `mock` subpackage

- Keep hand-written fakes in `mock`; expose function fields and invocation flags. Do not use mock generators or libraries.

### `main` package

- Put each executable in `cmd/<binary>`. Keep `main` small: harden first, reject unsupported arguments, wire dependencies, run the application, and use the central exit path.
- Prefer struct-literal injection unless a constructor is needed for validation or defaults.
- No package may define `init()`; it runs before process hardening.

## Security

### Platform and deployment

- The only supported target is Linux, and the intended deployment is TailsOS, an amnesic live system that runs from RAM, uses no persistent storage, and wipes memory on shutdown. That deployment is part of the security posture and is why the TTY is allowed as the display channel.
- Process hardening uses Linux facilities through the standard `syscall` package. Other operating systems are unsupported until equivalent hardening exists.

### Threat model

- **Trusted:** the operator, root, the kernel, and the hardware. A compromised kernel, root account, or hypervisor defeats every control here.
- **Defended against:** other non-root processes and users on the same host, the semi-trusted entropy sources, and anything that can observe OS-level channels.
- **Out of scope:** physical access, cold-boot and memory-remanence attacks (mitigated by TailsOS memory wiping), microarchitectural side channels such as Spectre, and terminal emulator compromise beyond what TailsOS's amnesia covers.
- The containment principle governs the running program. The build host necessarily persists source, the Go build cache, and test artifacts, so release builds come from a clean, documented, ideally ephemeral environment (such as a TailsOS session), with caches wiped afterward.

### Dependencies and toolchain

- `go.mod` MUST NOT contain `require` or `replace` directives. The standard library is the entire supply chain.
- Use only tools that ship with the Go toolchain (`go build`, `go vet`, `go test`). Do not run `go get`, `go install`, or `go mod download`, and do not add third-party linters or generators.
- Pin the toolchain with the `go` and `toolchain` directives in `go.mod`. Build with `GOTOOLCHAIN=local GOPROXY=off GOFLAGS=-mod=readonly` so the `go` command never touches the network.
- `CGO_ENABLED=0` always. `import "C"` is forbidden.
- These standard library imports are forbidden in all code, including tests:

  | Forbidden import | Reason |
  | --- | --- |
  | `net` and everything under it | No network. |
  | `os/exec` | No child processes. |
  | `plugin` | No dynamic code. |
  | `expvar`, `runtime/pprof`, `runtime/trace` | Diagnostic exporters. |
  | `log`, `log/syslog`, `log/slog` | No logging. |
  | `math/rand`, `math/rand/v2` | Non-cryptographic randomness. |
  | `unsafe` | Allowed only with explicit user approval and a written justification. |

### Build

- The release build is `CGO_ENABLED=0 GOTOOLCHAIN=local GOPROXY=off go build -trimpath -buildvcs=false -ldflags='-s -w' ./cmd/...`.
  - `-trimpath` removes build-host paths from the binary and from runtime stack traces.
  - `-buildvcs=false` keeps repository metadata out of the binary.
  - `-ldflags='-s -w'` strips symbol tables and DWARF data. It does **not** stop Go from printing stack traces; those are handled by process hardening.
- Release builds MUST be reproducible: two independent builds from the same source, toolchain, flags, and target produce identical hashes. Audit release binaries with `go version -m`.

### Process hardening

The hardening subpackage runs these steps first in `main`, before any input is read or any secret is touched, in this exact order. Except for the `/proc` reads in steps 10 and 11, no I/O happens between them. If any step fails, the program MUST exit with a failure code and produce no output (fail closed).

1. **Refuse runtime environment variables.** Exit if any `GO*` runtime variable is set (`GOTRACEBACK`, `GODEBUG`, `GOGC`, `GOMEMLIMIT`, `GOMAXPROCS`, and similar). They change runtime behavior and diagnostic output before `main` starts. Pin any needed settings at build time with `//go:debug` directives in `main`. Reading the environment for this check is the only allowed environment read.
2. **Silence stdout and stderr.** Point file descriptors 1 and 2 at `/dev/null` with `syscall.Dup3`, so accidental writes and anything the Go runtime prints on its own, such as fatal errors and panic traces, go nowhere.
3. **Not dumpable.** `prctl(PR_SET_DUMPABLE, 0)` via `syscall.RawSyscall(syscall.SYS_PRCTL, ...)`, defining constants locally if `syscall` lacks them. This blocks core dumps and same-user `ptrace`, and hides `/proc/<pid>/mem`, `cmdline`, and `environ` from unprivileged processes.
4. **No new privileges.** `prctl(PR_SET_NO_NEW_PRIVS, 1)`.
5. **No core dumps.** `syscall.Setrlimit(RLIMIT_CORE, 0)`.
6. **Per-buffer locking.** `syscall.Mlockall` MUST NOT be used: it locks the whole Go address space and may exceed `RLIMIT_MEMLOCK`. This step makes no system call; lock each sensitive buffer with `syscall.Mlock` when allocated (see [Memory hygiene](#memory-hygiene)). Runtime-created copies may remain unlocked; TailsOS having no swap is the primary protection.
7. **Minimal tracebacks.** `debug.SetTraceback("none")`. Step 1 is required because this call cannot go below the level set by `GOTRACEBACK`.
8. **Signals.** Catch `SIGQUIT`, `SIGINT`, `SIGTERM`, and `SIGHUP` with `os/signal` and route them to the central exit path. Go's default `SIGQUIT` handler dumps every goroutine's stack.
9. **Umask.** `syscall.Umask(0077)`.
10. **No debugger attached.** Read `/proc/self/status` and exit if `TracerPid` is not 0. Defense in depth behind step 3.
11. **Terminal scan.** Read `/proc` and exit if a process outside the program's ancestry holds the controlling pty open. Repeat the scan before every TTY read and before displaying source bytes or revealing the mnemonic. A scan is only a snapshot: another process can race it, and processes in the program's ancestry (such as `tmux`, `screen`, or `script`) are not detected. On a Linux virtual console, the scan is a no-op.

Then `main` opens the TTY (see [The TTY interface](#the-tty-interface)).

- `debug.SetCrashOutput` MUST NOT be used.
- **Startup window:** hardening cannot protect execution before step 1. Do no work before hardening.

### The TTY interface

The TTY is the program's only channel to and from its operator. It carries the interactive screens, the operator's typed input, the XOR display, and the one-time mnemonic-code reveal.

- Open the controlling terminal explicitly with `os.OpenFile("/dev/tty", os.O_RDWR, 0)` after hardening. It is the only input and output channel. Stdin (fd 0) is never read. `os.Stdout`, `os.Stderr`, `fmt.Print*`, and the builtins `print` and `println` MUST NOT be used; descriptors 1 and 2 point at `/dev/null`.
- Verify the descriptor is a real terminal with `ioctl(TCGETS)` via `syscall`. If there is no controlling terminal, fail closed. There is no batch, pipe, or file mode.
- The program MUST NOT change terminal settings (termios). The terminal stays in canonical mode with echo on, and input is read one line at a time.
- Every read from the TTY is bounded by the length of a locked line buffer. A line longer than the buffer is drained and rejected, never processed in pieces.
- The TTY MAY show only:
  1. fixed program text (warnings, menus, instructions, and error messages that contain no input data);
  2. operator input as echoed by the terminal, and the program's redisplay of input entered on the current screen;
  3. values derived from operator input, on the screen where that input was entered: the dice rolls entered so far and the SHA-256 digest they produce;
  4. the 32-byte dice contribution, 32-byte Linux kernel RNG contribution, and their 32-byte XOR result, as hexadecimal, together on the XOR screen after both sources succeed; the screen may be redrawn while active;
  5. the 24-word BIP-39 mnemonic code, exactly once, on the reveal screen after the terminal security check.

  Nothing else may be shown. In particular: no device bytes, no kernel-RNG or pool contents outside the XOR screen, and never the 24-word BIP-39 mnemonic code a second time.

- Operator-facing text MUST call the result the "24-word BIP-39 mnemonic code".
- Every buffer that holds items 2–5 before they are written (frame buffers, line buffers, XOR-display values) is sensitive memory and follows [Memory hygiene](#memory-hygiene).
- Output uses only these ANSI sequences: `ESC[?1049h` / `ESC[?1049l` (alternate screen on/off), `ESC[H` `ESC[2J` (home and clear), and `ESC[3J` (clear scrollback). Screen text is ASCII except for the Unicode Enter glyph `↵` (U+21B5) in command lists.
- TTY display is the one accepted weakening of the containment principle, justified by TailsOS: the terminal and its scrollback exist only in RAM and are wiped on shutdown. Ship operator guidance with the binary: run on TailsOS, in a fresh terminal, with no `tmux`, `screen`, `script`, or serial-console logging.
- Clearing the screen on exit is allowed as a courtesy but is not a security control; scrollback belongs to the terminal emulator.

### Exit paths

- All exits MUST go through one function that wipes sensitive memory and then calls `os.Exit` with a fixed code. `os.Exit` skips deferred calls, so wiping cannot rely on `defer`.
- Panic values MUST NOT contain sensitive data because the runtime may print them.
- Recover panics at the top of every goroutine the program starts, and convert them to the fixed failure exit. Unrecoverable runtime errors (concurrent map writes, out of memory) cannot be intercepted; steps 2 and 7 limit what they reveal.
- Exit codes are a small, fixed set defined as named constants. They identify a failure category, never data.
- Partial results and intermediate values stay in process memory and are wiped on every exit path.

### Runtime inputs and external I/O

- Arguments are visible through `/proc/<pid>/cmdline` and `ps`. Secrets and entropy MUST NOT be passed as arguments; this program accepts no arguments.
- Secrets and security-relevant configuration MUST NOT be read from environment variables.
- Do not create temporary files or directories (including on `tmpfs`, which can be swapped), and do not persist anything to disk.
- Do not start child processes, open sockets, or perform DNS lookups.

### Cryptography

- Use standard-library cryptography only. Use its implementation of a construction rather than writing your own.
- All randomness comes from `crypto/rand`.
- Operations on secret data MUST be constant-time: use `crypto/subtle` (`XORBytes`, `ConstantTimeCompare`, and related functions), with no branches, early returns, or table lookups indexed by secret data.
- MD5 and SHA-1 MUST NOT be used.
- Integer encoding uses `encoding/binary` with an explicit byte order and fixed-size types.

### Code clarity

- Security-sensitive calculations MUST be small, straightforward, and written for human review rather than performance. Avoid clever optimizations and secret-indexed lookups; isolate constant-time helpers and explain them.
- Document each calculation's input and output formats, algorithm, constant-time rationale (or why it is unnecessary), and a worked example. Keep comments consistent with operator-facing terminology.
- Test each calculation with deterministic cases covering its documented examples.

### Memory hygiene

- Store sensitive data in byte slices or fixed-size arrays, never strings. Allocate each buffer once at its final size on the heap, then immediately lock it with `syscall.Mlock` before writing to it; fail closed if locking fails. Do not append to sensitive buffers or call `Munlock`; locks are page-granular and released at process exit.
- Wipe buffers with `clear(b[:cap(b)])` as soon as possible, then call `runtime.KeepAlive(b)`. Assume copies may remain in moved stacks or garbage-collected memory; TailsOS having no swap is the primary protection. Keep sensitive allocations small.
- Do not format sensitive data or include it in errors. Error values describe fixed failure categories. Sensitive types MUST NOT expose contents through `String`, `GoString`, or `Format` methods.
- Channels MAY carry sensitive data only with ownership transfer: the receiver wipes it and the sender stops using it.

### Input handling

- Treat external input as untrusted. Bound reads and validate length and format before processing. Reject invalid input with a fixed response; never include it in program messages.
- Never read unbounded input or call `io.ReadAll` without a strict limit.

### Testing

- Use hardcoded deterministic fixtures, never real entropy or secrets. Tests MUST NOT write sensitive data to output or files, including through `t.Log`.
- Run tests with `-race`.

## Verification

Run these from the repository root before finishing any code change:

```sh
GOTOOLCHAIN=local GOPROXY=off go vet ./...
GOTOOLCHAIN=local GOPROXY=off go test -race ./...
go list -f '{{join .Imports "\n"}}{{"\n"}}{{join .TestImports "\n"}}{{"\n"}}{{join .XTestImports "\n"}}' ./... \
  | grep -E '^(C|log|net|net/.*|os/exec|plugin|expvar|runtime/pprof|runtime/trace|log/syslog|log/slog|math/rand|math/rand/v2|unsafe)$'   # must print nothing
grep -E '^\s*(require|replace)\b' go.mod   # must print nothing
```

These checks cover only this module's direct imports. The standard library's own internal dependencies (the runtime uses `unsafe`, for example) are not violations.

## Review checklist

1. Build with the release command and audit release binaries with `go version -m`.
2. Confirm there are no forbidden imports or module dependencies.
3. Confirm sensitive data is confined to the allowed TTY screens, stored and wiped safely, and never formatted or included in errors.
4. Review sensitive calculations for constant-time behavior and deterministic tests; run the test suite with `-race`.
5. Confirm hardening runs first, failures use the central wipe-and-exit path, and no package defines `init()`.
