package msgfee

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/coin"
	"github.com/iov-one/weave/errors"
)

var _ weave.QueryHandler = (*AntiSpamQuery)(nil)

// AntiSpamQuery allows querying currently set anti-spam fee
type AntiSpamQuery struct {
	minFee coin.Coin
}

func NewAntiSpamQuery(minFee coin.Coin) *AntiSpamQuery {
	return &AntiSpamQuery{minFee: minFee}
}

func (q *AntiSpamQuery) Query(db weave.ReadOnlyKVStore, mod string, data []byte) ([]weave.Model, error) {
	bytes, err := q.minFee.Marshal()
	if err != nil {
		return nil, errors.Wrap(err, "unable to marshal anti-spam fee")
	}
	return []weave.Model{weave.Pair([]byte(""), bytes)}, nil
}

func (q *AntiSpamQuery) RegisterQuery(qr weave.QueryRouter) {
	qr.Register("/minfee", q)
}
