package application

import (
	"bytes"
	"testing"
)

func TestXorInto(t *testing.T) {
	tests := []struct {
		name string
		dst  byte
		src  byte
		want byte
	}{
		{"01 xor 02", 0x01, 0x02, 0x03},
		{"00 xor ff", 0x00, 0xff, 0xff},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			dst := bytes.Repeat([]byte{test.dst}, 32)
			src := bytes.Repeat([]byte{test.src}, 32)
			xorInto(dst, src)
			if !bytes.Equal(dst, bytes.Repeat([]byte{test.want}, 32)) {
				t.Errorf("XOR result did not match %s fixture", test.name)
			}
		})
	}
}
