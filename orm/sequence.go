package orm

import (
	"encoding/binary"

	"github.com/iov-one/weave"
)

var seqPrefix = []byte("_s.")

// Sequence maintains a counter, and generates a
// series of keys. Each key is greater than the last,
// both NextInt() as well as bytes.Compare() on NextVal().
type Sequence struct {
	id []byte
}

// NewSequence creates a sequence with this id
// Form _s.<bucket>:<name>
// KeyTagger uses _s.<bucket> as key
func NewSequence(bucket, name string) Sequence {
	suffix := bucket + ":" + name
	return Sequence{
		id: append(seqPrefix, suffix...),
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

func (s *Sequence) curVal(db weave.KVStore) (key, val []byte) {
	key = append(seqPrefix, s.id...)
	val = db.Get(key)
	return key, val
}

func (s *Sequence) increment(db weave.KVStore) (int64, []byte) {
	key, bz := s.curVal(db)
	val := decodeSequence(bz)
	val++
	bz = encodeSequence(val)
	db.Set(key, bz)
	return val, bz
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
