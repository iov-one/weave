package auth

import (
	"bytes"

	"github.com/confio/weave/errors"
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
		return ErrInvalidSequence("Negative")
	}
	if seq == 0 && s.PubKey == nil {
		return ErrMissingPubKey()
	}
	if s.PubKey == nil && s.Address == nil {
		return ErrMissingPubKey()
	}
	if s.PubKey != nil && s.Address != nil {
		if !bytes.Equal(s.Address, s.PubKey.Address()) {
			return ErrPubKeyAddressMismatch()
		}
	}

	if s.Signature == nil {
		return errors.ErrMissingSignature()
	}

	return nil
}
