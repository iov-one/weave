package multisig

// MultiSigTx is an optional interface for a Tx that allows
// it to support multisig contract
type MultiSigTx interface {
	GetMultisig() [][]byte
}
