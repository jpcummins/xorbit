# Release verification and reproduction

Official releases provide a Linux/amd64 (`GOAMD64=v1`) executable. Each
release contains:

- `xorbit-linux-amd64` — the executable;
- `SHA256SUMS` — its SHA-256 checksum;
- `SHA256SUMS.asc` — the maintainer's detached GPG signature of the checksum
  manifest;
- `release-key.asc` — the maintainer's public GPG key.

The expected primary-key fingerprint is:

```text
01AC C456 133B A0C5 75CA F3EA AECE 715F 746C F11A
```

For first-time verification, compare this fingerprint with a copy obtained
through an independently trusted source. Do not trust a key only because it
was downloaded beside the signature it verifies.

## Verify a downloaded release

Download the four files above into one directory. Check that the public key's
primary fingerprint matches the expected fingerprint, then import it and
verify the signature and checksum:

```sh
gpg --show-keys --with-fingerprint release-key.asc
gpg --import release-key.asc
gpg --verify SHA256SUMS.asc SHA256SUMS
sha256sum --check SHA256SUMS
```

Proceed only if the key fingerprint is correct, GPG reports a good signature
from that key, and `sha256sum` reports `xorbit-linux-amd64: OK`. A good
signature proves that the imported key signed the manifest; the independent
fingerprint check establishes that it is the expected key.

## Rebuild and compare

Use the exact source tag for the release and Go 1.27.3. After checking out the
tag, put the downloaded `xorbit-linux-amd64` in the root of the source
checkout, alongside `go.mod`. Then build a second copy and compare the bytes:

```sh
git clone https://github.com/jpcummins/xorbit.git
cd xorbit
git checkout vX.Y.Z
git tag -v vX.Y.Z
go version

CGO_ENABLED=0 GOTOOLCHAIN=local GOPROXY=off GOFLAGS=-mod=readonly \
  GOOS=linux GOARCH=amd64 GOAMD64=v1 \
  go build -trimpath -buildvcs=false -ldflags='-s -w' \
  -o xorbit.rebuilt ./cmd/xorbit

cmp xorbit-linux-amd64 xorbit.rebuilt
```

Replace `vX.Y.Z` with the release tag. `go version` should report Go 1.27.3.
`cmp` produces no output when the two executables are byte-for-byte identical;
a difference means the builds do not match. Do not run a binary that fails
verification.

## Maintainer release procedure

1. Push the release source to `master` and wait for its checks to pass.
2. Create and push a signed, annotated version tag, for example:

   ```sh
   git tag -s vX.Y.Z -m "xorbit vX.Y.Z"
   git push origin vX.Y.Z
   ```

3. Wait for the tag workflow's `checks`, `build-a`, `build-b`, and
   `compare-builds` jobs to succeed. The two build jobs use separate GitHub
   runners and the same pinned Go version and target. Do not use an artifact
   from a failed workflow run.
4. Download and extract the `xorbit-linux-amd64-unsigned` Actions artifact. It
   contains the candidate executable, `SHA256SUMS`, and `release-key.asc`.
5. Independently build from that tag with the command above. Confirm the local
   executable matches the candidate with `cmp`, and confirm the manifest with
   `sha256sum --check SHA256SUMS`. Do not sign or publish a mismatch.
6. Sign the checksum manifest locally with the release key. The private key
   must not be uploaded to GitHub or placed in the workflow:

   ```sh
   gpg --armor --detach-sign \
     --local-user 01ACC456133BA0C575CAF3EAAECE715F746CF11A \
     --output SHA256SUMS.asc SHA256SUMS
   ```

7. Create a GitHub Release for the tag and upload the verified
   `xorbit-linux-amd64`, `SHA256SUMS`, `SHA256SUMS.asc`, and `release-key.asc`
   files. The Actions artifact is only a temporary candidate; the GitHub
   Release assets are the files users download and verify.
