package sigs

import (
	"github.com/iov-one/weave/errors"
)

// SignedTx represents a transaction that contains signatures,
// which can be verified by the auth.Decorator
type SignedTx interface {
	// GetSignBytes returns the canonical byte representation of the Msg.
	// Equivalent to weave.MustMarshal(tx.GetMsg()) if Msg has a deterministic
	// serialization.
	//
	// Helpful to store original, unparsed bytes here, just in case.
	GetSignBytes() ([]byte, error)

	// Signatures returns the signature of signers who signed the Msg.
	GetSignatures() []*StdSignature
}

// Validate ensures the StdSignature meets basic standards
func (s *StdSignature) Validate() error {
	seq := s.GetSequence()
	if seq < 0 {
		return errors.Wrap(ErrInvalidSequence, "negative")
	}
	if s.Pubkey == nil {
		return errors.Wrap(errors.ErrUnauthorized, "missing public key")
	}
	if s.Signature == nil {
		return errors.Wrap(errors.ErrUnauthorized, "missing signature")
	}

	return nil
}
