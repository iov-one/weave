package orm

import "github.com/iov-one/weave"

type IDGenerator interface {
	NextVal(db weave.KVStore, obj CloneableData) []byte
}

type IDGenBucket struct {
	Bucket
	idGen IDGenerator
}

func WithSeqIDGenerator(b Bucket, seqName string) IDGenBucket {
	seq := b.Sequence(seqName)
	return WithIDGenerator(b, IDGeneratorFunc(func(db weave.KVStore, _ CloneableData) []byte {
		return seq.NextVal(db)
	}))
}

func WithIDGenerator(b Bucket, gen IDGenerator) IDGenBucket {
	return IDGenBucket{
		Bucket: b,
		idGen:  gen,
	}
}

func (b IDGenBucket) Create(db weave.KVStore, data CloneableData) (Object, error) {
	id := b.idGen.NextVal(db, data)
	obj := NewSimpleObj(id, data)
	return obj, b.Save(db, obj)
}

type IDGeneratorFunc func(db weave.KVStore, obj CloneableData) []byte

func (i IDGeneratorFunc) NextVal(db weave.KVStore, obj CloneableData) []byte {
	return i(db, obj)
}
