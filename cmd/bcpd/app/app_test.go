package app

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/iov-one/weave/x/multisig"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	abci "github.com/tendermint/abci/types"
	"github.com/tendermint/tmlibs/log"

	"github.com/iov-one/weave/app"
	"github.com/iov-one/weave/crypto"
	"github.com/iov-one/weave/x"
	"github.com/iov-one/weave/x/cash"
	"github.com/iov-one/weave/x/namecoin"
	"github.com/iov-one/weave/x/sigs"
)

func TestApp(t *testing.T) {
	// no minimum fee, in-memory data-store
	chainID := "test-net-22"
	abciApp, err := GenerateApp("", log.NewNopLogger(), true)
	require.NoError(t, err)
	myApp := abciApp.(app.BaseApp)

	// let's set up a genesis file with some cash
	pk := crypto.GenPrivKeyEd25519()
	addr := pk.PublicKey().Address()

	appState := fmt.Sprintf(`{
        "wallets": [{
            "name": "demote",
            "address": "%s",
            "coins": [{
                "whole": 50000,
                "ticker": "ETH"
            },{
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
	}`, addr)

	// Commit first block, make sure non-nil hash
	myApp.InitChain(abci.RequestInitChain{AppStateBytes: []byte(appState), ChainId: chainID})
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
	checkWalletQuery(t, qres, "demote", 2, map[int]int64{0: 50000}, map[int]string{1: "FRNK"})

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
	dres, cres := signAndCommit(t, myApp, tx, []Signer{{pk, 0}}, chainID, 2)
	block2 := cres.Data
	assert.NotEmpty(t, block2)
	assert.NotEqual(t, block1, block2)

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

	// Query for new balances (same query, new state)
	qres = myApp.Query(query)
	checkWalletQuery(t, qres, "demote", 2, map[int]int64{0: 48000, 1: 1234}, nil)

	// make sure money arrived safely
	key2 := namecoin.NewWalletBucket().DBKey(addr2)
	query2 := abci.RequestQuery{
		Path: "/",
		Data: key2,
	}
	qres2 := myApp.Query(query2)
	checkWalletQuery(t, qres2, "", 1, map[int]int64{0: 2000}, map[int]string{0: "ETH"})

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

	// testing multisig contract
	// first create a contract
	// then a wallet at the contract address
	// finaly send money from the wallet controlled by contractAddr

	// create contract
	recovery1 := crypto.GenPrivKeyEd25519()
	recovery2 := crypto.GenPrivKeyEd25519()
	recovery3 := crypto.GenPrivKeyEd25519()
	signers := [][]byte{
		recovery1.PublicKey().Address(),
		recovery2.PublicKey().Address(),
		recovery3.PublicKey().Address()}
	cmsg := &multisig.CreateContractMsg{
		Sigs:                signers,
		ActivationThreshold: 2,
		AdminThreshold:      3, // immutable
	}
	tx = &Tx{
		Sum: &Tx_CreateContractMsg{cmsg},
	}
	dres, cres = signAndCommit(t, myApp, tx, []Signer{{pk, 1}}, chainID, 3)
	assert.NotEmpty(t, cres.Data)

	// retrieve contract ID
	recoveryContract := dres.Data

	// get a contract
	cquery := abci.RequestQuery{
		Path: "/contracts",
		Data: recoveryContract,
	}
	var c multisig.Contract
	cqres := myApp.Query(cquery)
	err = app.UnmarshalOneResult(cqres.Value, &c)
	require.NoError(t, err)
	assert.Equal(t, signers, c.Sigs)
	assert.EqualValues(t, 2, c.ActivationThreshold)
	assert.EqualValues(t, 3, c.AdminThreshold)

	// create master contract
	masterKey := crypto.GenPrivKeyEd25519()
	signers = [][]byte{
		masterKey.PublicKey().Address(),
		recoveryContract,
	}
	cmsg = &multisig.CreateContractMsg{
		Sigs:                signers,
		ActivationThreshold: 1,
		AdminThreshold:      2, // immutable
	}
	tx = &Tx{
		Sum: &Tx_CreateContractMsg{cmsg},
	}
	dres, cres = signAndCommit(t, myApp, tx, []Signer{{pk, 2}}, chainID, 4)
	assert.NotEmpty(t, cres.Data)

	// retrieve contract ID
	safeKeyContract := dres.Data
	safeKeyContractAddr := multisig.MultiSigCondition(safeKeyContract).Address()

	// get a contract
	cquery = abci.RequestQuery{
		Path: "/contracts",
		Data: safeKeyContract,
	}
	var c1 multisig.Contract
	cqres = myApp.Query(cquery)
	err = app.UnmarshalOneResult(cqres.Value, &c1)
	require.NoError(t, err)
	assert.Equal(t, signers, c1.Sigs)
	assert.EqualValues(t, 1, c1.ActivationThreshold)
	assert.EqualValues(t, 2, c1.AdminThreshold)

	// create a wallet at contractAddr
	msg = &cash.SendMsg{
		Src:  addr,
		Dest: safeKeyContractAddr,
		Amount: &x.Coin{
			Whole:  2000,
			Ticker: "ETH",
		},
		Memo: "New wallet controlled by a contract",
	}
	tx = &Tx{
		Sum: &Tx_SendMsg{msg},
	}
	_, cres = signAndCommit(t, myApp, tx, []Signer{{pk, 3}}, chainID, 5)
	assert.NotEmpty(t, cres.Data)

	// build and sign a transaction using master key to activate safeKeyContract
	msg = &cash.SendMsg{
		Src:  safeKeyContractAddr,
		Dest: addr2,
		Amount: &x.Coin{
			Whole:  1000,
			Ticker: "ETH",
		},
		Memo: "Gift from a contract!",
	}
	tx = &Tx{
		Sum:      &Tx_SendMsg{msg},
		Multisig: [][]byte{safeKeyContract},
	}
	_, cres = signAndCommit(t, myApp, tx, []Signer{{masterKey, 0}}, chainID, 6)
	assert.NotEmpty(t, cres.Data)

	// make sure money arrived safely
	cwquery := abci.RequestQuery{
		Path: "/",
		Data: key2,
	}
	cwqres := myApp.Query(cwquery)
	checkWalletQuery(t, cwqres, "", 1, map[int]int64{0: 3000}, map[int]string{0: "ETH"})

	// Now do the same operation but using recoveryContract to activate safeKeyContract
	msg = &cash.SendMsg{
		Src:  safeKeyContractAddr,
		Dest: addr2,
		Amount: &x.Coin{
			Whole:  1000,
			Ticker: "ETH",
		},
		Memo: "Gift from a contract!",
	}
	tx = &Tx{
		Sum:      &Tx_SendMsg{msg},
		Multisig: [][]byte{recoveryContract, safeKeyContract},
	}
	_, cres = signAndCommit(t, myApp, tx, []Signer{{recovery1, 0}, {recovery2, 0}}, chainID, 7)
	assert.NotEmpty(t, cres.Data)

	// make sure money arrived safely
	cwquery = abci.RequestQuery{
		Path: "/",
		Data: key2,
	}
	cwqres = myApp.Query(cwquery)
	checkWalletQuery(t, cwqres, "", 1, map[int]int64{0: 4000}, map[int]string{0: "ETH"})
}

type Signer struct {
	pk    *crypto.PrivateKey
	nonce int64
}

// signAndCommit signs tx with signatures from signers and submits to the chain
// asserts and fails the test in case of errors during the process
func signAndCommit(t *testing.T, app app.BaseApp, tx *Tx, signers []Signer, chainID string, blockHeight int64) (abci.ResponseDeliverTx, abci.ResponseCommit) {
	for _, signer := range signers {
		sig, err := sigs.SignTx(signer.pk, tx, chainID, signer.nonce)
		require.NoError(t, err)
		tx.Signatures = append(tx.Signatures, sig)
	}

	txBytes, err := tx.Marshal()
	require.NoError(t, err)
	require.NotEmpty(t, txBytes)

	// Submit to the chain
	header := abci.Header{Height: blockHeight}
	app.BeginBlock(abci.RequestBeginBlock{Header: header})
	// check and deliver must pass
	chres := app.CheckTx(txBytes)
	require.Equal(t, uint32(0), chres.Code, chres.Log)

	dres := app.DeliverTx(txBytes)
	require.Equal(t, uint32(0), dres.Code, dres.Log)

	app.EndBlock(abci.RequestEndBlock{})
	cres := app.Commit()
	return dres, cres
}

// checkWalletQuery checks the results of a wallet query along with the received wallet
// maps are used to the wallet state eg. {0: 50000}, {1:"FRNK"} would assert that the first coin whole is 50000 and
// the second coin ticker is "ETH" in a wallet of at least 2 coins
func checkWalletQuery(t *testing.T, res abci.ResponseQuery, name string, nbCoins int, wholes map[int]int64, tickers map[int]string) {
	// check query was ok
	require.Equal(t, uint32(0), res.Code, "%#v", res)
	assert.NotEmpty(t, res.Value)

	var w namecoin.Wallet
	err := app.UnmarshalOneResult(res.Value, &w)
	require.NoError(t, err)
	require.Equal(t, nbCoins, len(w.Coins))

	for idx, whole := range wholes {
		assert.True(t, len(w.Coins) > idx)
		assert.Equal(t, whole, w.Coins[idx].Whole)
	}

	for idx, ticker := range tickers {
		assert.True(t, len(w.Coins) > idx)
		assert.Equal(t, ticker, w.Coins[idx].Ticker)
	}
}
