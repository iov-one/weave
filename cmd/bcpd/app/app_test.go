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
	queryAndCheckWallet(t, myApp, "/", key,
		namecoin.Wallet{
			Name: "demote",
			Coins: x.Coins{
				{
					Ticker: "ETH",
					Whole:  50000,
				},
				{
					Ticker: "FRNK",
					Whole:  1234,
				},
			},
		})

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
	queryAndCheckWallet(t, myApp, "/", key,
		namecoin.Wallet{
			Name: "demote",
			Coins: x.Coins{
				{
					Ticker: "ETH",
					Whole:  48000,
				},
				{
					Ticker: "FRNK",
					Whole:  1234,
				},
			},
		})

	// make sure money arrived safely
	key2 := namecoin.NewWalletBucket().DBKey(addr2)
	queryAndCheckWallet(t, myApp, "/", key2,
		namecoin.Wallet{
			Coins: x.Coins{
				{
					Ticker: "ETH",
					Whole:  2000,
				},
			},
		})

	// make sure other paths also get this value....
	queryAndCheckWallet(t, myApp, "/wallets", addr2,
		namecoin.Wallet{
			Coins: x.Coins{
				{
					Ticker: "ETH",
					Whole:  2000,
				},
			},
		})

	// make sure other paths also get this value....
	queryAndCheckWallet(t, myApp, "/wallets?prefix", addr2[:15],
		namecoin.Wallet{
			Coins: x.Coins{
				{
					Ticker: "ETH",
					Whole:  2000,
				},
			},
		})

	// and we can query by name (sender account)
	queryAndCheckWallet(t, myApp, "/wallets/name", []byte("demote"),
		namecoin.Wallet{
			Name: "demote",
			Coins: x.Coins{
				{
					Ticker: "ETH",
					Whole:  48000,
				},
				{
					Ticker: "FRNK",
					Whole:  1234,
				},
			},
		})

	// get a token
	queryAndCheckToken(t, myApp, "/tokens", []byte("ETH"),
		[]namecoin.Token{
			{
				Name:    "Smells like ethereum",
				SigFigs: int32(9),
			},
		})

	// get all tokens
	queryAndCheckToken(t, myApp, "/tokens?prefix", nil,
		[]namecoin.Token{
			{
				Name:    "Smells like ethereum",
				SigFigs: int32(9),
			},
			{
				Name:    "Frankie",
				SigFigs: int32(3),
			},
		})

	// create recoveryContract
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
	queryAndCheckContract(t, myApp, "/contracts", recoveryContract,
		multisig.Contract{
			Sigs:                signers,
			ActivationThreshold: 2,
			AdminThreshold:      3,
		})

	// create safeKeyContract contract
	// can be activated by masterKey or recoveryContract
	masterKey := crypto.GenPrivKeyEd25519()
	signers = [][]byte{
		masterKey.PublicKey().Address(),
		multisig.MultiSigCondition(recoveryContract).Address(),
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
	queryAndCheckContract(t, myApp, "/contracts", safeKeyContract,
		multisig.Contract{
			Sigs:                signers,
			ActivationThreshold: 1,
			AdminThreshold:      2,
		})

	// create a wallet controlled by safeKeyContract
	safeKeyContractAddr := multisig.MultiSigCondition(safeKeyContract).Address()
	msg = &cash.SendMsg{
		Src:  addr,
		Dest: safeKeyContractAddr,
		Amount: &x.Coin{
			Whole:  2000,
			Ticker: "ETH",
		},
		Memo: "New wallet controlled by safeKeyContract",
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
	queryAndCheckWallet(t, myApp, "/", key2,
		namecoin.Wallet{
			Coins: x.Coins{
				{
					Ticker: "ETH",
					Whole:  3000,
				},
			},
		})

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
	queryAndCheckWallet(t, myApp, "/", key2,
		namecoin.Wallet{
			Coins: x.Coins{
				{
					Ticker: "ETH",
					Whole:  4000,
				},
			},
		})
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

func queryAndCheckWallet(t *testing.T, baseApp app.BaseApp, path string, data []byte, expected namecoin.Wallet) {
	query := abci.RequestQuery{Path: path, Data: data}
	res := baseApp.Query(query)

	// check query was ok
	require.Equal(t, uint32(0), res.Code, "%#v", res)
	assert.NotEmpty(t, res.Value)

	var actual namecoin.Wallet
	err := app.UnmarshalOneResult(res.Value, &actual)
	require.NoError(t, err)
	require.Equal(t, expected, actual)
}

func queryAndCheckContract(t *testing.T, baseApp app.BaseApp, path string, data []byte, expected multisig.Contract) {
	query := abci.RequestQuery{Path: path, Data: data}
	res := baseApp.Query(query)

	// check query was ok
	require.Equal(t, uint32(0), res.Code, "%#v", res)
	assert.NotEmpty(t, res.Value)

	var actual multisig.Contract
	err := app.UnmarshalOneResult(res.Value, &actual)
	require.NoError(t, err)
	require.Equal(t, expected, actual)
}

func queryAndCheckToken(t *testing.T, baseApp app.BaseApp, path string, data []byte, expected []namecoin.Token) {
	query := abci.RequestQuery{Path: path, Data: data}
	res := baseApp.Query(query)

	var set app.ResultSet
	err := set.Unmarshal(res.Value)
	require.NoError(t, err)
	assert.Equal(t, len(expected), len(set.Results))

	for i, obj := range set.Results {
		var actual namecoin.Token
		err = actual.Unmarshal(obj)
		require.NoError(t, err)
		require.Equal(t, expected[i], actual)
	}
}
