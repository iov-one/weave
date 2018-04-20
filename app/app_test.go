package app

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/iov-one/bcp-demo/x/namecoin"
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
	myApp := abciApp.(app.BaseApp)

	// let's set up a genesis file with some cash
	pk := crypto.GenPrivKeyEd25519()
	addr := pk.PublicKey().Address()
	genesis := fmt.Sprintf(`{
        "chain_id": "%s",
        "app_state": {
            "wallets": [{
                "name": "demote",
                "address": "%s",
                "coins": [{
                    "whole": 50000,
                    "ticker": "ETH"
                    }, {
                    "whole": 1234,
                    "ticker": "FRNK"
                }]
            }],
            "tokens": [{
                "ticker": "ETH",
                "name": "Smells like ethereum",
                "sig_figs": 9
            },{
                "ticker": "FRNK",
                "name": "Frankie",
                "sig_figs": 3
            }]
        }
    }`, chainID, addr)

	// Commit first block, make sure non-nil hash
	myApp.InitChainWithGenesis(abci.RequestInitChain{}, []byte(genesis))
	header := abci.Header{Height: 1}
	myApp.BeginBlock(abci.RequestBeginBlock{Header: header})
	myApp.EndBlock(abci.RequestEndBlock{})
	cres := myApp.Commit()
	block1 := cres.Data
	assert.NotEmpty(t, block1)
	assert.Equal(t, chainID, myApp.GetChainID())

	// Query for my balance
	key := namecoin.NewWalletBucket().DBKey(addr)
	query := abci.RequestQuery{
		Path: "/",
		Data: key,
	}
	qres := myApp.Query(query)
	require.Equal(t, uint32(0), qres.Code, "%#v", qres)
	assert.NotEmpty(t, qres.Value)
	// parse it and check it is not empty
	var acct namecoin.Wallet
	err = app.UnmarshalOneResult(qres.Value, &acct)
	require.NoError(t, err)
	require.Equal(t, 2, len(acct.Coins))
	assert.Equal(t, "demote", acct.Name)
	assert.Equal(t, int64(50000), acct.Coins[0].Whole)
	assert.Equal(t, "FRNK", acct.Coins[1].Ticker)

	// build and sign a transaction
	pk2 := crypto.GenPrivKeyEd25519()
	addr2 := pk2.PublicKey().Address()
	msg := &cash.SendMsg{
		Src:  addr,
		Dest: addr2,
		Amount: &x.Coin{
			Whole:  2000,
			Ticker: "ETH",
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
	myApp.BeginBlock(abci.RequestBeginBlock{Header: header})
	// check and deliver must pass
	chres := myApp.CheckTx(txBytes)
	require.Equal(t, uint32(0), chres.Code, chres.Log)
	dres := myApp.DeliverTx(txBytes)
	require.Equal(t, uint32(0), dres.Code, dres.Log)

	// ensure 3 keys with proper values
	if assert.Equal(t, 3, len(dres.Tags), "%#v", dres.Tags) {
		// three keys we expect, in order
		keys := make([][]byte, 3)
		vals := [][]byte{[]byte("s"), []byte("s"), []byte("s")}
		hexWllt := []byte("776C6C743A")
		hexSigs := []byte("736967733A")
		keys[0] = append(hexSigs, []byte(addr.String())...)
		keys[1] = append(hexWllt, []byte(addr.String())...)
		keys[2] = append(hexWllt, []byte(addr2.String())...)
		if bytes.Compare(addr2, addr) < 0 {
			keys[1], keys[2] = keys[2], keys[1]
		}
		// make sure the DeliverResult matches expections
		assert.Equal(t, keys[0], dres.Tags[0].Key)
		assert.Equal(t, keys[1], dres.Tags[1].Key)
		assert.Equal(t, keys[2], dres.Tags[2].Key)
		assert.Equal(t, vals[0], dres.Tags[0].Value)
		assert.Equal(t, vals[1], dres.Tags[1].Value)
		assert.Equal(t, vals[2], dres.Tags[2].Value)
	}

	// make sure commit is proper
	myApp.EndBlock(abci.RequestEndBlock{})
	// commit should produce a different hash
	cres = myApp.Commit()
	block2 := cres.Data
	assert.NotEmpty(t, block2)
	assert.NotEqual(t, block1, block2)

	// Query for new balances (same query, new state)
	qres = myApp.Query(query)
	require.Equal(t, uint32(0), qres.Code, "%#v", qres)
	assert.NotEmpty(t, qres.Value)
	// parse it and check it is not empty
	var acct2 namecoin.Wallet
	err = app.UnmarshalOneResult(qres.Value, &acct2)
	require.NoError(t, err)
	assert.Equal(t, "demote", acct2.Name)
	require.Equal(t, 2, len(acct2.Coins))
	assert.Equal(t, int64(48000), acct2.Coins[0].Whole)
	assert.Equal(t, int64(1234), acct2.Coins[1].Whole)

	// make sure money arrived safely
	key2 := namecoin.NewWalletBucket().DBKey(addr2)
	query2 := abci.RequestQuery{
		Path: "/",
		Data: key2,
	}
	qres2 := myApp.Query(query2)
	require.Equal(t, uint32(0), qres2.Code, "%#v", qres2)
	// parse it and check it is not empty
	var acct3 namecoin.Wallet
	err = app.UnmarshalOneResult(qres2.Value, &acct3)
	require.NoError(t, err)
	require.Equal(t, 1, len(acct3.Coins))
	assert.Equal(t, int64(2000), acct3.Coins[0].Whole)
	assert.Equal(t, "ETH", acct3.Coins[0].Ticker)

	// make sure other paths also get this value....
	query3 := abci.RequestQuery{
		Path: "/wallets",
		Data: addr2,
	}
	qres3 := myApp.Query(query3)
	require.Equal(t, uint32(0), qres3.Code, "%#v", qres3)
	assert.Equal(t, qres2.Key, qres3.Key)
	assert.Equal(t, qres2.Value, qres3.Value)

	// make sure other paths also get this value....
	query4 := abci.RequestQuery{
		Path: "/wallets?prefix",
		Data: addr2[:15],
	}
	qres4 := myApp.Query(query4)
	require.Equal(t, uint32(0), qres4.Code, "%#v", qres4)
	assert.Equal(t, qres2.Key, qres4.Key)
	assert.Equal(t, qres2.Value, qres4.Value)

	// and we can query by name (sender account)
	query5 := abci.RequestQuery{
		Path: "/wallets/name",
		Data: []byte("demote"),
	}
	qres5 := myApp.Query(query5)
	require.Equal(t, uint32(0), qres5.Code, "%#v", qres5)
	assert.Equal(t, qres.Key, qres5.Key)
	assert.Equal(t, qres.Value, qres5.Value)

	// get a token
	tquery := abci.RequestQuery{
		Path: "/tokens",
		Data: []byte("ETH"),
	}
	var toke namecoin.Token
	tres := myApp.Query(tquery)
	err = app.UnmarshalOneResult(tres.Value, &toke)
	require.NoError(t, err)
	assert.Equal(t, int32(9), toke.SigFigs)
	assert.Equal(t, "Smells like ethereum", toke.Name)

	// get all tokens
	aquery := abci.RequestQuery{
		Path: "/tokens?prefix",
	}
	ares := myApp.Query(aquery)
	var set app.ResultSet
	err = set.Unmarshal(ares.Value)
	require.NoError(t, err)
	assert.Equal(t, 2, len(set.Results))
	err = toke.Unmarshal(set.Results[1])
	require.NoError(t, err)
	assert.Equal(t, int32(3), toke.SigFigs)
	assert.Equal(t, "Frankie", toke.Name)
}
