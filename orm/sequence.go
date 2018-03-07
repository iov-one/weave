package orm

import (
	"encoding/binary"

	"github.com/confio/weave"
)

var seqPrefix = []byte("_s:")

// Sequence maintains a counter/auto-generate a number of
// keys, they may be sequential or pseudo-random,
// but must be deterministic.
type Sequence struct {
	id []byte
}

// NewSequence creates a sequence with this id
func NewSequence(id []byte) Sequence {
	return Sequence{
		id: id,
	}
}

// NextVal increments the sequence and returns next val as 8 bytes
func (s *Sequence) NextVal(db weave.KVStore) []byte {
	_, bz := s.increment(db)
	return bz
}

// NextInt increments the sequence and returns next val as int
func (s *Sequence) NextInt(db weave.KVStore) int64 {
	val, _ := s.increment(db)
	return val
}

func (s *Sequence) increment(db weave.KVStore) (int64, []byte) {
	key := append(seqPrefix, s.id...)
	bz := db.Get(key)
	val := decodeSequence(bz)
	val++
	bz = encodeSequence(val)
	return val, bz
}

func decodeSequence(bz []byte) int64 {
	if bz == nil {
		return 0
	}
	val := binary.LittleEndian.Uint64(bz)
	return int64(val)
}

func encodeSequence(val int64) []byte {
	bz := make([]byte, 8)
	binary.LittleEndian.PutUint64(bz, uint64(val))
	return bz
}
