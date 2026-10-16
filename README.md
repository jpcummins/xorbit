# xorbit

`xorbit` creates a 24-word BIP-39 mnemonic code by combining dice
entropy with 256 bits from the Linux kernel RNG.

## Why

The idea behind `xorbit` is to avoid relying on just one source of entropy. It
combines dice rolls with 256 bits from the Linux kernel RNG using XOR. The July
2026 ColdCard entropy hack prompted this project and highlighted the value of
combining independent entropy sources to strengthen security.

## Installation

### Download a release

Prebuilt releases target Linux/amd64 (`GOAMD64=v1`) and are published on the
[GitHub Releases page](https://github.com/jpcummins/xorbit/releases). Verify
the GPG signature and checksum before running a downloaded executable. See
[RELEASES.md](RELEASES.md) for verification and reproduction instructions.

The expected primary GPG fingerprint is
`01AC C456 133B A0C5 75CA F3EA AECE 715F 746C F11A`. Confirm it through a
trusted source independent of the release download before trusting the key.

### Build from source

Use Go 1.27.1 and build the release target from the repository root:

```sh
CGO_ENABLED=0 GOTOOLCHAIN=local GOPROXY=off GOFLAGS=-mod=readonly \
  GOOS=linux GOARCH=amd64 GOAMD64=v1 \
  go build -trimpath -buildvcs=false -ldflags='-s -w' \
  -o xorbit ./cmd/xorbit
```

The explicit Go version and target, `CGO_ENABLED=0`, and offline module
settings make the build inputs repeatable. `-trimpath` removes local build
paths, `-buildvcs=false` omits repository metadata, and `-ldflags='-s -w'`
removes symbol and debug tables. The release workflow builds the tagged source
on two independent runners and requires identical binary hashes. See
[RELEASES.md](RELEASES.md) to compare a local build with the published binary.

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
