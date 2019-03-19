package sigs

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
)

// NextNonce returns the next numeric nonce value that should be used during a
// transaction signing.
// Any address can contain a nonce. In practice you always want to acquire a
// nonce for the signer. You can get the signers address by calling
//   address := <crypto.Signer>.PublicKey().Address()
func NextNonce(db weave.ReadOnlyKVStore, signer weave.Address) (int64, error) {
	obj, err := NewBucket().Get(db, signer)
	if err != nil {
		return 0, errors.Wrap(err, "bucket get")
	}
	if u := AsUser(obj); u != nil {
		return u.Sequence, nil
	}

	// If not yet present, nonce counting starts with zero.
	return 0, nil
}
