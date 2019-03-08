package weavetest

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/crypto"
)

// NewKey returns a newly generated unique private key.
func NewKey() crypto.Signer {
	return crypto.GenPrivKeyEd25519()
}

// NewCondition returns a newly generated unique weave condition.
// To create a weave address call Address method of returned condition.
func NewCondition() weave.Condition {
	return NewKey().PublicKey().Condition()
}
