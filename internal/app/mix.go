package application

import (
	"crypto/subtle"
	"runtime"
)

// xorInto computes the 32-byte XOR of src into dst. Each input and the output
// are exactly 32 locked bytes. For each position i, it writes dst[i] XOR
// src[i] back to dst[i] with crypto/subtle. The fixtures 0x01 XOR 0x02 = 0x03
// and 0x00 XOR 0xff = 0xff show the byte operation. Work and memory access
// do not depend on the secret byte values.
func xorInto(dst, src []byte) {
	if len(dst) != 32 || len(src) != 32 {
		panic("xorInto: both buffers must be 32 bytes")
	}
	subtle.XORBytes(dst, dst, src)
	runtime.KeepAlive(dst)
	runtime.KeepAlive(src)
}
