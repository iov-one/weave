package multisig

// MultiSigTx is an optional interface for a Tx that allows it to
// support multisig contract. Multisig authentication can be done only
// for transactions that do support this interface.
type MultiSigTx interface {
	GetMultisig() [][]byte
}
