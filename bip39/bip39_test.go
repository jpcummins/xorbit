package bip39

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"strings"
	"testing"

	app "xorbit/internal/app"
)

var _ app.WordLookup = Lookup{}

func entropy(hexText string) []byte {
	data, err := hex.DecodeString(hexText)
	if err != nil || len(data) != 32 {
		panic("bad deterministic entropy fixture")
	}
	return data
}

func TestEnglishFileHash(t *testing.T) {
	data, err := os.ReadFile("english.txt")
	if err != nil {
		t.Fatal("could not read embedded English wordlist fixture")
	}
	sum := sha256.Sum256(data)
	const want = "2f5eed53a4727b4bf8880d8f3f199efc90e58503646d9ff8eff3a2ed3b24dbda"
	if hex.EncodeToString(sum[:]) != want {
		t.Fatal("English wordlist fixture hash does not match BIP-39")
	}
}

func TestWordlistHash(t *testing.T) {
	var rebuilt strings.Builder
	var word [8]byte
	lookup := Lookup{}
	for index := 0; index < 2048; index++ {
		lookup.WriteWord(word[:], uint16(index))
		rebuilt.Write(bytes.TrimRight(word[:], " "))
		if index < 2047 {
			rebuilt.WriteByte('\n')
		}
	}
	rebuilt.WriteByte('\n')
	sum := sha256.Sum256([]byte(rebuilt.String()))
	const want = "2f5eed53a4727b4bf8880d8f3f199efc90e58503646d9ff8eff3a2ed3b24dbda"
	if hex.EncodeToString(sum[:]) != want {
		t.Fatal("embedded wordlist does not match the official BIP-39 file")
	}
}

func TestWriteWordFixtures(t *testing.T) {
	tests := []struct {
		name  string
		index uint16
		want  string
	}{
		{"first", 0, "abandon "},
		{"last", 2047, "zoo     "},
		{"middle", 1330, "play    "},
	}
	lookup := Lookup{}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var word [8]byte
			lookup.WriteWord(word[:], test.index)
			if string(word[:]) != test.want {
				t.Errorf("WriteWord(%d) did not match its fixture", test.index)
			}
		})
	}
}

func TestWordIndexFixtures(t *testing.T) {
	tests := []struct {
		name     string
		entropy  string
		position int
		want     uint16
	}{
		{"all-zero first", strings.Repeat("00", 32), 0, 0},
		{"all-ones first", strings.Repeat("ff", 32), 0, 2047},
		{"all-ones last regular", strings.Repeat("ff", 32), 22, 2047},
		{"all-zero last regular", strings.Repeat("00", 32), 22, 0},
		{"all-7f first", strings.Repeat("7f", 32), 0, 1019},
		{"all-7f second", strings.Repeat("7f", 32), 1, 2015},
		{"all-80 first", strings.Repeat("80", 32), 0, 1028},
		{"all-80 second", strings.Repeat("80", 32), 1, 32},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := wordIndex(entropy(test.entropy), test.position)
			if got != test.want {
				t.Errorf("word index did not match fixture: got %d, want %d", got, test.want)
			}
		})
	}
}

func TestChecksumByteFixtures(t *testing.T) {
	tests := []struct {
		name    string
		entropy string
		want    byte
	}{
		{"all-zero", strings.Repeat("00", 32), 0x66},
		{"all-ones", strings.Repeat("ff", 32), 0xaf},
		{"all-7f", strings.Repeat("7f", 32), 0x17},
		{"all-80", strings.Repeat("80", 32), 0xbd},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := checksumByte(entropy(test.entropy), make([]byte, 32)); got != test.want {
				t.Errorf("checksum byte did not match fixture: got %#x, want %#x", got, test.want)
			}
		})
	}
}

func TestChecksumWordIndexFixtures(t *testing.T) {
	tests := []struct {
		name    string
		entropy string
		want    uint16
	}{
		{"all-zero", strings.Repeat("00", 32), 102},
		{"all-ones", strings.Repeat("ff", 32), 1967},
		{"all-7f", strings.Repeat("7f", 32), 1815},
		{"all-80", strings.Repeat("80", 32), 189},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := checksumWordIndex(entropy(test.entropy), make([]byte, 32)); got != test.want {
				t.Errorf("checksum word index did not match fixture: got %d, want %d", got, test.want)
			}
		})
	}
}

func TestMnemonicWordIndexFixtures(t *testing.T) {
	tests := []struct {
		name     string
		entropy  string
		position int
		want     uint16
	}{
		{"first word of zero entropy", strings.Repeat("00", 32), 0, 0},
		{"word 24 of zero entropy", strings.Repeat("00", 32), 23, 102},
		{"first word of all-ones entropy", strings.Repeat("ff", 32), 0, 2047},
		{"word 24 of all-ones entropy", strings.Repeat("ff", 32), 23, 1967},
	}
	lookup := Lookup{}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := lookup.MnemonicWordIndex(entropy(test.entropy), test.position, make([]byte, 32))
			if got != test.want {
				t.Errorf("mnemonic word index did not match fixture: got %d, want %d", got, test.want)
			}
		})
	}
}
