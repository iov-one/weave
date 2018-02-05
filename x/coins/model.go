package coins

import (
	"github.com/confio/weave"
)

//---- Key

// Key is the primary key we use to distinguish users
// This should be []byte, in order to index with our KVStore.
// Any structure to these bytes should be defined by the constructor.
//
// Question: allow objects with a Marshal method???
type Key []byte

var walletPrefix = []byte("wallet:")

// NewKey constructs the coin key from a key hash,
// by appending a prefix.
func NewKey(addr weave.Address) Key {
	bz := append(walletPrefix, addr...)
	return Key(bz)
}
