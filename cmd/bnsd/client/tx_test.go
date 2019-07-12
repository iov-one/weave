package client

import (
	"testing"

	"github.com/iov-one/weave/coin"
	"github.com/iov-one/weave/migration"
	"github.com/iov-one/weave/store"
	"github.com/iov-one/weave/weavetest/assert"
	"github.com/iov-one/weave/x/cash"
	"github.com/iov-one/weave/x/sigs"
)

func TestSendTx(t *testing.T) {
	source := GenPrivateKey()
	sourceAddr := source.PublicKey().Address()
	rcpt := GenPrivateKey().PublicKey().Address()
	amount := coin.Coin{Whole: 59, Fractional: 42, Ticker: "ECK"}

	chainID := "ding-dong"
	tx := BuildSendTx(sourceAddr, rcpt, amount, "Hi There")
	// if we sign with 0, we can validate against an empty db
	SignTx(tx, source, chainID, 0)

	// make sure the tx has a sig
	assert.Equal(t, 1, len(tx.GetSignatures()))

	// make sure this validates
	db := store.MemStore()
	migration.MustInitPkg(db, "sigs")
	conds, err := sigs.VerifyTxSignatures(db, tx, chainID)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(conds))
	assert.Equal(t, source.PublicKey().Condition(), conds[0])

	// make sure other chain doesn't validate
	db = store.MemStore()
	_, err = sigs.VerifyTxSignatures(db, tx, "foobar")
	assert.Equal(t, true, err != nil)

	// parse tx and verify we have the proper fields
	data, err := tx.Marshal()
	assert.Nil(t, err)
	parsed, err := ParseBcpTx(data)
	assert.Nil(t, err)
	msg, err := parsed.GetMsg()
	assert.Nil(t, err)
	send, ok := msg.(*cash.SendMsg)
	assert.Equal(t, true, ok)

	assert.Equal(t, "Hi There", send.Memo)
	assert.Equal(t, rcpt, send.Destination)
	assert.Equal(t, sourceAddr, send.Source)
	assert.Equal(t, int64(59), send.Amount.Whole)
	assert.Equal(t, "ECK", send.Amount.Ticker)
}
