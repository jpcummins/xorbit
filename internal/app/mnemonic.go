package application

// WordLookup provides only the BIP-39 operations used by the App. WriteWord
// writes directly into an 8-byte destination so mnemonic words need not pass
// through a temporary string or array.
type WordLookup interface {
	// WriteWord writes the 8-byte padded English word for index 0..2047.
	WriteWord(dst []byte, index uint16)
	// MnemonicWordIndex returns the index for position 0..23 from 32-byte
	// entropy, using 32 locked scratch bytes for the checksum calculation.
	MnemonicWordIndex(entropy []byte, position int, scratch []byte) uint16
}

// mnemonicWordIndex asks the injected BIP-39 lookup for one word index and
// wipes its locked checksum scratch before returning the secret-derived index.
func (a *Runner) mnemonicWordIndex(position int) uint16 {
	scratch := a.WS.Scratch[:32:32]
	index := a.Words.MnemonicWordIndex(a.WS.Pool, position, scratch)
	wipe(scratch)
	return index
}
