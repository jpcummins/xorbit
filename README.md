# xorbit

`xorbit` creates a 24-word BIP-39 mnemonic code by combining dice
entropy with 256 bits from the Linux kernel RNG.

## Why

The idea behind `xorbit` is to avoid relying on just one source of entropy. It
combines dice rolls with 256 bits from the Linux kernel RNG using XOR. The July
2026 ColdCard entropy hack prompted this project and highlighted the value of
combining independent entropy sources to strengthen security.

## Installation

The recommended way to build `xorbit` is with the security-hardening settings below. From the repository root, run:

```sh
CGO_ENABLED=0 GOTOOLCHAIN=local GOPROXY=off GOFLAGS=-mod=readonly \
  go build -trimpath -buildvcs=false -ldflags='-s -w' \
  -o xorbit ./cmd/xorbit
```

These settings make the release build self-contained and reproducible:

- `CGO_ENABLED=0` builds without C dependencies.
- `GOTOOLCHAIN=local` uses the installed, pinned Go toolchain; `GOPROXY=off`
  disables network access, and `GOFLAGS=-mod=readonly` prevents module-file
  changes.
- `-trimpath` removes local build paths, and `-buildvcs=false` omits version
  control metadata.
- `-ldflags='-s -w'` removes symbol and debug tables to reduce binary size.
- `-o xorbit` names the output executable.

## Usage

Run the program in a fresh, private terminal:

```sh
./xorbit
```

Have fair six-sided dice, pen, and paper ready. Follow the on-screen
instructions. The program requires a controlling terminal and takes no
arguments.

## Security

`xorbit` handles sensitive entropy and a 24-word BIP-39 mnemonic code, so I’ve
added safeguards to reduce the risk of that information escaping the program.
It hardens the process before reading input, silences standard output and
error, disables core dumps, and locks and clears sensitive buffers. It creates
no data files, opens no network connections, and starts no child processes.

See [SECURITY.md](SECURITY.md) for precautions, limitations, and vulnerability
reporting.

## License

BSD 3-Clause. See [LICENSE](LICENSE).
