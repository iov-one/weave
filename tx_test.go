package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/confio/weave/store"
	"github.com/confio/weave/x"
	"github.com/confio/weave/x/cash"
	"github.com/confio/weave/x/sigs"
)

func TestSendTx(t *testing.T) {
	sender := GenPrivateKey()
	senderAddr := sender.PublicKey().Address()
	rcpt := GenPrivateKey().PublicKey().Address()
	amount := x.Coin{Whole: 59, Fractional: 42, Ticker: "ECK"}

	chainID := "ding-dong"
	tx := BuildSendTx(senderAddr, rcpt, amount, "Hi There")
	// if we sign with 0, we can validate against an empty db
	SignTx(tx, sender, chainID, 0)

	// make sure the tx has a sig
	require.Equal(t, 1, len(tx.GetSignatures()))

	// make sure this validates
	db := store.MemStore()
	conds, err := sigs.VerifyTxSignatures(db, tx, chainID)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(conds))
	assert.EqualValues(t, sender.PublicKey().Condition(), conds[0])

	// make sure other chain doesn't validate
	db = store.MemStore()
	_, err = sigs.VerifyTxSignatures(db, tx, "foobar")
	assert.Error(t, err)

	// parse tx and verify we have the proper fields
	data, err := tx.Marshal()
	require.NoError(t, err)
	parsed, err := ParseBcpTx(data)
	require.NoError(t, err)
	msg, err := parsed.GetMsg()
	require.NoError(t, err)
	send, ok := msg.(*cash.SendMsg)
	require.True(t, ok)

	assert.Equal(t, "Hi There", send.Memo)
	assert.EqualValues(t, rcpt, send.Dest)
	assert.EqualValues(t, senderAddr, send.Src)
	assert.Equal(t, int64(59), send.Amount.Whole)
	assert.Equal(t, "ECK", send.Amount.Ticker)
}
