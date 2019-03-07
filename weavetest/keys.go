package weavetest

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/crypto"
)

func NewKey() crypto.Signer {
	return crypto.GenPrivKeyEd25519()
}

func NewCondition() weave.Condition {
	return NewKey().PublicKey().Condition()
}
