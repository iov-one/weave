package approvals

// MultiSigTx is an optional interface for a Tx that allows
// it to support multisig contract
type ApprovalTx interface {
	GetApproval() [][]byte
}
