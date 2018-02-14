package ideas

import (
	"github.com/confio/weave"
	"github.com/stretchr/testify/assert"
)

// DB contains multiple tables in one store
type DB struct {
	kv weave.KVStore
}

// Bucket is a prefixed subspace of the DB
// proto defines the default Model, all elements of this type
type Bucket struct {
	name  string
	proto Creator
}

// Creator creates a new model in memory.
// It can call Clone on prototypical model
type Creator interface {
	Create() Model
}

// Get one element
func (t Bucket) Get(db DB, key Keyed) (Model, error) {
	// TODO
	return nil, nil
}

// Save will write a model, it must be of the same type as proto
func (t Bucket) Save(db DB, model Model) error {
	return nil
}

//--- TODO: Bucket ----
//
// Composite primary index (valigator, ) - One or List
//  -> Special by type
// Secondary index (eg. ByName) - One or List
//  -> How to store this?
//
// table:<bucket>:<key> -> Model
// index:<bucket>:<name>:<key> -> primary key
//
// Write query functions for each specific type, but these are all
// one-liners that just delegate and type-cast

//--- TODO: Sequence ----
//
// Set up an incremental counter, set up for one bucket
// TODO: LastVal?

// Sequence maintains a counter/auto-generate a number of
// keys, they may be sequential or pseudo-random
type Sequence struct {
	string id
}

// NextVal is a new primary key
func (s *Sequence) NextVal(db DB) []byte {

}

// Model is what is stored in the bucket
// Key is joined with the prefix to set the full key
// Value is the data stored
//
// this can be light wrapper around a protobuf-defined type
type Model interface {
	weave.Persistent
	// TODO: key must be set explicitly, or auto-computed?
	// Add helper for auto-sequence
	Keyed
	Value() interface{}
}

// Keyed is anything that can identify itself
type Keyed interface {
	Key() []byte
	SetKey([]byte) // should only be called if Key() returns nil
}

func demo() {
	db := MockDB()
	addr := weave.NewAddress([]byte("foo"))

	// TODO: wrap with strongly typed wrapper that exposed *BlogPost instead of Model)
	bucket := SequentialBucket{
		sequence: "blog",
		bucket: Bucket{
			name:  "blogs",
			proto: BlogPost{Title: "Hello, world"},
		},
	}.WithDB(db)

	var first *BlogPost = bucket.Create()
	first.Author = addr
	first.Body = "This is my first post"
	bucket.Save(first)

	second := bucket.Create()
	second.Author = addr
	second.Title = "Some special text"
	bucket.Save(second)

	// load by one specific key
	load := bucket.Get(second.Key())
	assert.Equal(load.Title, "Some special text")

	// iterate over composite primary key
	mine := bucket.ByAddress(addr)
	assert.True(mine.Valid())
	assert.Equal(first, mine.Value())
	mine.Next()
	assert.True(mine.Valid())
	assert.Equal(second, mine.Value())
	mine.Next()
	assert.False(mine.Valid())

	// TODO: secondary index by Title (StartsWith)
	es := bucket.WithTitlePrefix("S")
	assert.Equal(1, len(es.AsList()))

	alpha := bucket.ByTitle()
	assert.Equal(2, len(alpha.AsList()))

	answer := bucket.WithTitleBetween("No", "Yes")
	assert.True(answer.Valid())
	assert.Equal(second, answer.Value())
	answer.Next()
	assert.True(answer.Valid())
	//...

}

// TODO: build wrappers with functionality
// Add Queue as refinement of Table
