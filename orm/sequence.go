package orm

import (
	"encoding/binary"

	"github.com/iov-one/weave"
)

// Sequence maintains a counter, and generates a
// series of keys. Each key is greater than the last,
// both NextInt() as well as bytes.Compare() on NextVal().
type Sequence struct {
	id []byte
}

// NewSequence returns a sequence counter. Sequence is using following pattern
// to construct a key:
//    _s.<bucket>:<name>
func NewSequence(bucket, name string) Sequence {
	id := "_s." + bucket + ":" + name
	return Sequence{
		id: []byte(id),
	}
}

// NextVal increments the sequence and returns its state as 8 bytes.
func (s *Sequence) NextVal(db weave.KVStore) []byte {
	_, bz := s.increment(db, 1)
	return bz
}

// NextInt increments the sequence and returns its state as int.
func (s *Sequence) NextInt(db weave.KVStore) int64 {
	val, _ := s.increment(db, 1)
	return val
}

func (s *Sequence) increment(db weave.KVStore, inc int64) (int64, []byte) {
	raw := db.Get(s.id)
	val := decodeSequence(raw)
	val += inc
	raw = encodeSequence(val)
	db.Set(s.id, raw)
	return val, raw
}

func decodeSequence(bz []byte) int64 {
	if bz == nil {
		return 0
	}
	val := binary.BigEndian.Uint64(bz)
	return int64(val)
}

func encodeSequence(val int64) []byte {
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, uint64(val))
	return bz
}
