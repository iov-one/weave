package app

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	abci "github.com/tendermint/abci/types"
	"github.com/tendermint/tmlibs/log"

	"github.com/confio/weave"
	"github.com/confio/weave/app"
	"github.com/confio/weave/crypto"
	"github.com/confio/weave/x"
	"github.com/confio/weave/x/cash"
	"github.com/confio/weave/x/sigs"
)

func testInitChain(t *testing.T, myApp app.BaseApp, addr string) {
	// initialize chain
	appState := fmt.Sprintf(`{
            "cash": [{
                "address": "%s",
                "coins": [{
                    "whole": 50000,
                    "ticker": "ETH"
                    }, {
                    "whole": 1234,
                    "ticker": "FRNK"
                }]
            }]}`, addr)
	assert.Equal(t, "", myApp.GetChainID())
	myApp.InitChain(abci.RequestInitChain{AppStateBytes: []byte(appState)})
}

// testCommit will commit at height h and return new hash
func testCommit(t *testing.T, myApp app.BaseApp, h int64, chainId string) []byte {
	// Commit first block, make sure non-nil hash
	header := abci.Header{Height: h, ChainID: chainId}
	myApp.BeginBlock(abci.RequestBeginBlock{Header: header})
	assert.Equal(t, chainId, myApp.GetChainID())
	myApp.EndBlock(abci.RequestEndBlock{})
	cres := myApp.Commit()
	hash := cres.Data
	assert.NotEmpty(t, hash)
	return hash
}

func testQuery(t *testing.T, myApp app.BaseApp, path string, key []byte, obj weave.Persistent) {
	// Query for my balance
	query := abci.RequestQuery{
		Path: path,
		Data: key,
	}
	qres := myApp.Query(query)
	require.Equal(t, uint32(0), qres.Code, "%#v", qres)
	assert.NotEmpty(t, qres.Value)
	if path == "/" {
		// the original key will be embedded in a result set
		// this should add two bytes to it
		assert.Equal(t, len(key)+2, len(qres.Key), "%x", qres.Key)
	}
	// unpack the ResultSet
	// parse it and check it is not empty
	err := app.UnmarshalOneResult(qres.Value, obj)
	require.NoError(t, err)
}

func testSendTx(t *testing.T, myApp app.BaseApp, h int64,
	amount int64, ticker string,
	sender *crypto.PrivateKey, rcpt weave.Address, seq int64) abci.ResponseDeliverTx {

	msg := &cash.SendMsg{
		Src:  sender.PublicKey().Address(),
		Dest: rcpt,
		Amount: &x.Coin{
			Whole:  amount,
			Ticker: ticker,
		},
		Memo: "Have a great trip!",
	}
	tx := &Tx{
		Sum: &Tx_SendMsg{msg},
	}
	sig, err := sigs.SignTx(sender, tx, myApp.GetChainID(), seq)
	require.NoError(t, err)
	tx.Signatures = []*sigs.StdSignature{sig}
	txBytes, err := tx.Marshal()
	require.NoError(t, err)
	require.NotEmpty(t, txBytes)

	// Submit to the chain
	header := abci.Header{Height: h}
	myApp.BeginBlock(abci.RequestBeginBlock{Header: header})
	// check and deliver must pass
	chres := myApp.CheckTx(txBytes)
	require.Equal(t, uint32(0), chres.Code, chres.Log)
	dres := myApp.DeliverTx(txBytes)
	require.Equal(t, uint32(0), dres.Code, dres.Log)
	return dres
}

func TestApp(t *testing.T) {
	// no minimum fee, in-memory data-store
	chainID := "test-net-22"
	abciApp, err := GenerateApp("", log.NewNopLogger())
	require.NoError(t, err)
	myApp := abciApp.(app.BaseApp)

	// let's set up a genesis file with some cash
	pk := crypto.GenPrivKeyEd25519()
	addr := pk.PublicKey().Address()

	testInitChain(t, myApp, addr.String())
	hash1 := testCommit(t, myApp, 1, chainID)

	var acct cash.Set
	key := cash.NewBucket().DBKey(addr)
	testQuery(t, myApp, "/", key, &acct)
	require.Equal(t, 2, len(acct.Coins))
	assert.Equal(t, int64(50000), acct.Coins[0].Whole)
	assert.Equal(t, "FRNK", acct.Coins[1].Ticker)

	// build and sign a transaction
	pk2 := crypto.GenPrivKeyEd25519()
	addr2 := pk2.PublicKey().Address()
	dres := testSendTx(t, myApp, 2, 2000, "ETH", pk, addr2, 0)
	// and commit the block
	hash2 := testCommit(t, myApp, 2, chainID)
	assert.NotEqual(t, hash1, hash2)

	// ensure 3 keys with proper values
	if assert.Equal(t, 3, len(dres.Tags), "%#v", dres.Tags) {
		// three keys we expect, in order
		keys := make([][]byte, 3)
		vals := [][]byte{[]byte("s"), []byte("s"), []byte("s")}
		hexCash := []byte("636173683A")
		hexSigs := []byte("736967733A")
		keys[0] = append(hexCash, []byte(addr.String())...)
		keys[1] = append(hexCash, []byte(addr2.String())...)
		keys[2] = append(hexSigs, []byte(addr.String())...)
		if bytes.Compare(addr2, addr) < 0 {
			keys[0], keys[1] = keys[1], keys[0]
		}
		// make sure the DeliverResult matches expections
		assert.Equal(t, keys[0], dres.Tags[0].Key)
		assert.Equal(t, keys[1], dres.Tags[1].Key)
		assert.Equal(t, keys[2], dres.Tags[2].Key)
		assert.Equal(t, vals[0], dres.Tags[0].Value)
		assert.Equal(t, vals[1], dres.Tags[1].Value)
		assert.Equal(t, vals[2], dres.Tags[2].Value)
	}

	// Query for new balances (same key, new state)
	var acct2 cash.Set
	testQuery(t, myApp, "/", key, &acct2)
	require.Equal(t, 2, len(acct2.Coins))
	assert.Equal(t, int64(48000), acct2.Coins[0].Whole)
	assert.Equal(t, int64(1234), acct2.Coins[1].Whole)

	// make sure money arrived safely
	var acct3 cash.Set
	key2 := cash.NewBucket().DBKey(addr2)
	testQuery(t, myApp, "/", key2, &acct3)
	require.Equal(t, 1, len(acct3.Coins))
	assert.Equal(t, int64(2000), acct3.Coins[0].Whole)
	assert.Equal(t, "ETH", acct3.Coins[0].Ticker)

	// make sure other paths also get this value....
	var acct4 cash.Set
	testQuery(t, myApp, "/wallets", addr2, &acct4)
	require.Equal(t, 1, len(acct4.Coins))
	assert.Equal(t, int64(2000), acct4.Coins[0].Whole)
	assert.Equal(t, "ETH", acct4.Coins[0].Ticker)

	// prefix scan works
	var acct5 cash.Set
	testQuery(t, myApp, "/wallets?prefix", addr, &acct5)
	require.Equal(t, 2, len(acct2.Coins))
	assert.Equal(t, int64(48000), acct2.Coins[0].Whole)
	assert.Equal(t, int64(1234), acct2.Coins[1].Whole)

	// try another send
	testSendTx(t, myApp, 3, 100, "FRNK", pk, addr2, 1)
	// and commit the block
	hash3 := testCommit(t, myApp, 3, chainID)
	assert.NotEqual(t, hash2, hash3)

	var second cash.Set
	testQuery(t, myApp, "/wallets", addr2, &second)
	require.Equal(t, 2, len(second.Coins))
	assert.Equal(t, int64(2000), second.Coins[0].Whole)
	assert.Equal(t, "ETH", second.Coins[0].Ticker)
	assert.Equal(t, int64(100), second.Coins[1].Whole)
	assert.Equal(t, "FRNK", second.Coins[1].Ticker)
}
