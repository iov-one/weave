package app

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	abci "github.com/tendermint/abci/types"
	"github.com/tendermint/tmlibs/log"

	"github.com/confio/weave/app"
	"github.com/confio/weave/crypto"
	"github.com/confio/weave/x"
	"github.com/confio/weave/x/cash"
	"github.com/confio/weave/x/sigs"
)

func TestApp(t *testing.T) {
	// no minimum fee, in-memory data-store
	chainID := "test-net-22"
	abciApp, err := GenerateApp("", log.NewNopLogger())
	require.NoError(t, err)
	app := abciApp.(app.BaseApp)

	// let's set up a genesis file with some cash
	pk := crypto.GenPrivKeyEd25519()
	addr := pk.PublicKey().Address()
	genesis := fmt.Sprintf(`{
        "chain_id": "%s",
        "app_state": {
            "cash": [{
                "address": "%X",
                "coins": [{
                    "integer": 50000,
                    "currency_code": "ETH"
                    }, {
                    "integer": 1234,
                    "currency_code": "FRNK"
                }]
            }]
        }
    }`, chainID, addr)

	// Commit first block, make sure non-nil hash
	app.InitChainWithGenesis(abci.RequestInitChain{}, []byte(genesis))
	header := abci.Header{Height: 1}
	app.BeginBlock(abci.RequestBeginBlock{Header: header})
	app.EndBlock(abci.RequestEndBlock{})
	cres := app.Commit()
	block1 := cres.Data
	assert.NotEmpty(t, block1)
	assert.Equal(t, chainID, app.GetChainID())

	// Query for my balance
	key := cash.NewKey(addr)
	query := abci.RequestQuery{
		Path: "/key",
		Data: key,
	}
	qres := app.Query(query)
	require.Equal(t, uint32(0), qres.Code, "%#v", qres)
	assert.NotEmpty(t, qres.Value)
	// parse it and check it is not empty
	var acct cash.Set
	err = acct.Unmarshal(qres.Value)
	require.NoError(t, err)
	require.Equal(t, 2, len(acct.Coins))
	assert.Equal(t, int32(50000), acct.Coins[0].Integer)
	assert.Equal(t, "FRNK", acct.Coins[1].CurrencyCode)

	// build and sign a transaction
	pk2 := crypto.GenPrivKeyEd25519()
	addr2 := pk2.PublicKey().Address()
	msg := &cash.SendMsg{
		Src:  addr,
		Dest: addr2,
		Amount: &x.Coin{
			Integer:      2000,
			CurrencyCode: "ETH",
		},
		Memo: "Have a great trip!",
	}
	tx := &Tx{
		Sum: &Tx_SendMsg{msg},
	}
	sig, err := sigs.SignTx(pk, tx, chainID, 0)
	require.NoError(t, err)
	tx.Signatures = []*sigs.StdSignature{sig}
	txBytes, err := tx.Marshal()
	require.NoError(t, err)
	require.NotEmpty(t, txBytes)

	// Submit to the chain
	header = abci.Header{Height: 2}
	app.BeginBlock(abci.RequestBeginBlock{Header: header})
	// check and deliver must pass
	chres := app.CheckTx(txBytes)
	require.Equal(t, uint32(0), chres.Code, chres.Log)
	dres := app.DeliverTx(txBytes)
	require.Equal(t, uint32(0), dres.Code, dres.Log)
	app.EndBlock(abci.RequestEndBlock{})
	// commit should produce a different hash
	cres = app.Commit()
	block2 := cres.Data
	assert.NotEmpty(t, block2)
	assert.NotEqual(t, block1, block2)

	// Query for new balances (same query, new state)
	qres = app.Query(query)
	require.Equal(t, uint32(0), qres.Code, "%#v", qres)
	assert.NotEmpty(t, qres.Value)
	// parse it and check it is not empty
	var acct2 cash.Set
	err = acct2.Unmarshal(qres.Value)
	require.NoError(t, err)
	require.Equal(t, 2, len(acct2.Coins))
	assert.Equal(t, int32(48000), acct2.Coins[0].Integer)
	assert.Equal(t, int32(1234), acct2.Coins[1].Integer)

	// make sure money arrived safely
	key2 := cash.NewKey(addr2)
	query2 := abci.RequestQuery{
		Path: "/key",
		Data: key2,
	}
	qres2 := app.Query(query2)
	require.Equal(t, uint32(0), qres2.Code, "%#v", qres2)
	// parse it and check it is not empty
	var acct3 cash.Set
	err = acct3.Unmarshal(qres2.Value)
	require.NoError(t, err)
	require.Equal(t, 1, len(acct3.Coins))
	assert.Equal(t, int32(2000), acct3.Coins[0].Integer)
	assert.Equal(t, "ETH", acct3.Coins[0].CurrencyCode)

}
