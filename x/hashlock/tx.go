package hashlock

import (
	"crypto/sha256"

	"github.com/confio/weave"
)

// HashKeyTx is an optional interface for a Tx that allows
// it to provide Keys (Preimages) to open HashLocks
type HashKeyTx interface {
	// GetPreimage should return a hash preimage if provided
	// or nil if not included in this tx
	GetPreimage() []byte
}

// PreimagePermission calculates a sha256 hash and then
func PreimagePermission(preimage []byte) weave.Permission {
	h := sha256.Sum256(preimage)
	return weave.NewPermission("hash", "sha256", h[:])
}
