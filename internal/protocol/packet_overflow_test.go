package protocol

import (
	"encoding/binary"
	"testing"
)

func TestDecodeRejectsOverflowLengths(t *testing.T) {
	for _, tag := range []byte{26, 42} {
		for _, length := range []uint64{1 << 63, ^uint64(0), 1000} {
			data := binary.AppendUvarint([]byte{tag}, length)
			if _, err := Decode(data); err == nil {
				t.Fatalf("accepted overflow length %d", length)
			}
		}
	}
}
