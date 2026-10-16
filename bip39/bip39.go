// Package bip39 owns the English wordlist and the BIP-39 word-index and
// checksum calculations. Lookup exposes only the operations used by the App.
package bip39

import (
	"crypto/sha256"
	"crypto/subtle"
	"runtime"

	app "xorbit/internal/app"
)

var _ app.WordLookup = Lookup{}

// Lookup performs constant-time BIP-39 word lookup and mnemonic index
// calculation. It has no state.
type Lookup struct{}

// WriteWord writes the English BIP-39 word for index into an 8-byte slot.
// The input is an index from 0..2047; dst is exactly 8 bytes. The output is
// ASCII, right-padded with spaces. It clears dst, scans all 2048 newline-
// separated words in englishTxt, and selects matching bytes with a
// constant-time mask. It never indexes the wordlist by index.
//
// Worked fixtures: index 0 writes "abandon ", index 2047 writes "zoo     ",
// and index 1330 writes "play    ". The loop reads every word and processes
// exactly 8 columns per word. Its wordlist cursor depends only on the public
// embedded text; the secret index affects only constant-time masks.
func (Lookup) WriteWord(dst []byte, index uint16) {
	if len(dst) != 8 {
		panic("bip39: word destination must be 8 bytes")
	}
	clear(dst)
	hi := uint8(index >> 8)
	lo := uint8(index)
	wordlistOffset := 0
	for i := 0; i < 2048; i++ {
		mask := byte(uint8(
			subtle.ConstantTimeByteEq(uint8(i>>8), hi)&
				subtle.ConstantTimeByteEq(uint8(i), lo),
		) * 0xff)
		for column := 0; column < 8; column++ {
			letter := byte(' ')
			if wordlistOffset < len(englishTxt) && englishTxt[wordlistOffset] != '\n' {
				letter = englishTxt[wordlistOffset]
				wordlistOffset++
			}
			dst[column] = (dst[column] &^ mask) | (letter & mask)
		}
		if wordlistOffset < len(englishTxt) && englishTxt[wordlistOffset] == '\n' {
			wordlistOffset++
		}
	}
	runtime.KeepAlive(dst)
}

// MnemonicWordIndex returns the BIP-39 word index at position 0..23 for a
// 32-byte entropy input. scratch is a 32-byte locked buffer used for the
// final-word checksum. Positions 0..22 extract 11 consecutive bits,
// most-significant first. Position 23 combines the entropy's last 3 bits
// with the first byte of SHA-256(entropy). Position is public, so selecting
// the final-word rule may branch on it; processing entropy does not branch
// on secret bits. The caller wipes scratch after this call.
//
// Worked fixtures: for all-zero entropy, position 0 is index 0 ("abandon")
// and position 23 is index 102 ("art"); for all-ones entropy, position 0
// is index 2047 ("zoo") and position 23 is index 1967 ("vote").
func (Lookup) MnemonicWordIndex(entropy []byte, position int, scratch []byte) uint16 {
	if len(entropy) != 32 || len(scratch) != 32 || position < 0 || position > 23 {
		panic("bip39: invalid entropy or word position")
	}
	if position == 23 {
		return checksumWordIndex(entropy, scratch)
	}
	return wordIndex(entropy, position)
}

// wordIndex extracts one 11-bit BIP-39 word index from a 32-byte entropy
// slice. Position is 0..22; the result is a big-endian 11-bit integer.
// Entropy bits are numbered most-significant first within each byte. The
// function shifts in exactly 11 bits and does not branch or look up data by
// secret bits.
//
// Worked fixtures: all-0x7f at position 0 gives 1019 ("legal"); all-0x80
// gives 1028 ("letter"); all-0x00 gives 0 ("abandon"); all-0xff gives
// 2047 ("zoo").
func wordIndex(entropy []byte, position int) uint16 {
	bit := position * 11
	var index uint16
	for offset := 0; offset < 11; offset++ {
		bitPosition := bit + offset
		byteIndex := bitPosition / 8
		bitOffset := uint(bitPosition % 8)
		bitValue := uint16((entropy[byteIndex] >> (7 - bitOffset)) & 1)
		index = (index << 1) | bitValue
	}
	return index
}

// checksumByte returns the first byte of SHA-256(entropy), where entropy and
// the locked scratch buffer are exactly 32 bytes. It writes the digest into
// scratch with hash.Hash.Sum, reads its first byte, then resets the hash state.
// SHA-256 processes every input byte with the same algorithm.
//
// Worked fixtures: all-0x00 -> 0x66, all-0xff -> 0xaf, all-0x7f -> 0x17,
// and all-0x80 -> 0xbd.
func checksumByte(entropy, scratch []byte) byte {
	if len(entropy) != 32 || len(scratch) != 32 {
		panic("bip39: checksum requires 32-byte buffers")
	}
	h := sha256.New()
	h.Write(entropy)
	sum := h.Sum(scratch[:0])
	checksum := sum[0]
	h.Reset()
	runtime.KeepAlive(scratch)
	return checksum
}

// checksumWordIndex returns word 24's 11-bit index from 32-byte entropy:
// the last 3 entropy bits form the high bits, followed by checksumByte, which
// writes the SHA-256 digest into locked scratch.
// The bit selection does not branch on secret bytes.
//
// Worked fixtures: all-zero entropy gives (0<<8)|0x66 = 102 ("art");
// all-0xff gives (7<<8)|0xaf = 1967 ("vote").
func checksumWordIndex(entropy, scratch []byte) uint16 {
	lastBits := uint16(entropy[31] & 0x07)
	return (lastBits << 8) | uint16(checksumByte(entropy, scratch))
}
