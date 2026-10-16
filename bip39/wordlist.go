package bip39

import _ "embed"

// english.txt is a verbatim copy of bitcoin/bips, bip-0039/english.txt:
// 2048 lines, one word per line, ending with a newline. SHA-256 of the file
// (verified by TestEnglishFileHash):
//
//	2f5eed53a4727b4bf8880d8f3f199efc90e58503646d9ff8eff3a2ed3b24dbda
//
// Why this is a separate file rather than a Go string literal: a 13 KB
// literal is hard to review, easy to typo, and inconvenient for security
// audits. Storing the canonical BIP-39 file byte-for-byte lets an
// independent reviewer run `diff` against the upstream source. The embed
// package pulls the file into the binary at build time, so the byte
// content reaches the program without runtime filesystem access (AGENTS.md
// "no files / no network"). No init() function is involved.
//
//go:embed english.txt
var englishTxt string
