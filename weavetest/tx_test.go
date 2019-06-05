package weavetest

import (
	"bytes"
	"testing"
)

func TestSequenceID(t *testing.T) {
	numToEnc := map[uint64][]byte{
		1:      {0, 0, 0, 0, 0, 0, 0, 1},
		2:      {0, 0, 0, 0, 0, 0, 0, 2},
		3:      {0, 0, 0, 0, 0, 0, 0, 3},
		4:      {0, 0, 0, 0, 0, 0, 0, 4},
		123:    {0, 0, 0, 0, 0, 0, 0, 123},
		123123: {0, 0, 0, 0, 0, 1, 224, 243},
	}
	for id, want := range numToEnc {
		got := SequenceID(uint64(id))
		if !bytes.Equal(want, got) {
			t.Fatalf("id=%d, want %d got %d", id, want, got)
		}
	}
}
