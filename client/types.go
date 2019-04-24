package client

import "github.com/iov-one/weave"

// TransactionID is the hash used to identify the transaction
type TransactionID []byte

// TxQuery is some query to find transactions
type TxQuery string

// MempoolResult is returned from the mempool (CheckTx)
// Result is only set on success codes, Err is set if it was a failure code
type MempoolResult struct {
	ID     TransactionID
	Result *weave.CheckResult
	Err    error
}

// AsCommitError will turn an errored MempoolResult into a CommitResult
func (a MempoolResult) AsCommitError() CommitResult {
	if a.Err == nil {
		panic("failed assertion: AsCommitError can onyl be called on errors")
	}
	return CommitResult{
		ID:  a.ID,
		Err: a.Err,
	}
}

// CommitResult is returned from the block (DeliverTx)
// Result is only set on success codes, Err is set if it was a failure code
type CommitResult struct {
	ID     TransactionID
	Height int64
	Result *weave.DeliverResult
	Err    error
}
