package app_test

import (
	"bytes"
	"testing"

	"github.com/iov-one/weave/x/multisig"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	weave_app "github.com/iov-one/weave/app"
	"github.com/iov-one/weave/cmd/bnsd/app"
	"github.com/iov-one/weave/cmd/bnsd/app/testdata/fixtures"
	"github.com/iov-one/weave/crypto"
	"github.com/iov-one/weave/x"
	"github.com/iov-one/weave/x/cash"
	"github.com/iov-one/weave/x/namecoin"
	"github.com/iov-one/weave/x/sigs"
	abci "github.com/tendermint/tendermint/abci/types"
)

func TestApp(t *testing.T) {
	appFixture := fixtures.NewApp()
	addr := appFixture.GenesisKeyAddress
	pk := appFixture.GenesisKey
	chainID := appFixture.ChainID
	myApp := appFixture.Build()
	// Query for my balance
	key := namecoin.NewWalletBucket().DBKey(appFixture.GenesisKeyAddress)
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
	dres := sendToken(t, myApp, appFixture.ChainID, 2, []Signer{{pk, 0}}, addr, addr2, 2000, "ETH", "Have a great trip!")

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
	recoveryContract := createContract(t, myApp, chainID, 3, []Signer{{pk, 1}},
		2, recovery1.PublicKey().Address(), recovery2.PublicKey().Address(), recovery3.PublicKey().Address())

	// create safeKeyContract contract
	// can be activated by masterKey or recoveryContract
	masterKey := crypto.GenPrivKeyEd25519()
	safeKeyContract := createContract(t, myApp, chainID, 4, []Signer{{pk, 2}},
		1, masterKey.PublicKey().Address(), multisig.MultiSigCondition(recoveryContract).Address())

	// create a wallet controlled by safeKeyContract
	safeKeyContractAddr := multisig.MultiSigCondition(safeKeyContract).Address()
	sendToken(t, myApp, chainID, 5, []Signer{{pk, 3}},
		addr, safeKeyContractAddr, 2000, "ETH", "New wallet controlled by safeKeyContract")

	// build and sign a transaction using master key to activate safeKeyContract
	receiver := crypto.GenPrivKeyEd25519()
	sendToken(t, myApp, chainID, 6, []Signer{{masterKey, 0}},
		safeKeyContractAddr, receiver.PublicKey().Address(), 1000, "ETH", "Gift from a contract!", safeKeyContract)

	// Now do the same operation but using recoveryContract to activate safeKeyContract
	// create a new receiver so it is easy to check its balance (no need to remember previous one)
	receiver = crypto.GenPrivKeyEd25519()
	sendToken(t, myApp, chainID, 7, []Signer{{recovery1, 0}, {recovery2, 0}},
		safeKeyContractAddr, receiver.PublicKey().Address(), 1000, "ETH", "Another gift from a contract!",
		recoveryContract, safeKeyContract)
}

type Signer struct {
	pk    *crypto.PrivateKey
	nonce int64
}

// sendToken creates the transaction, signs it and sends it
// checks money has arrived safely
func sendToken(t *testing.T, baseApp weave_app.BaseApp, chainID string, height int64, signers []Signer,
	from, to []byte, amount int64, ticker, memo string, contracts ...[]byte) abci.ResponseDeliverTx {
	msg := &cash.SendMsg{
		Src:  from,
		Dest: to,
		Amount: &x.Coin{
			Whole:  amount,
			Ticker: ticker,
		},
		Memo: memo,
	}

	tx := &app.Tx{
		Sum:     &app.Sum{&app.Sum_SendMsg{msg}},
		Multisig: contracts,
	}

	res := signAndCommit(t, baseApp, tx, signers, chainID, height)

	// make sure money arrived safely
	queryAndCheckWallet(t, baseApp, "/wallets", to,
		namecoin.Wallet{
			Coins: x.Coins{
				{
					Ticker: ticker,
					Whole:  amount,
				},
			},
		})

	return res
}

// createContract creates an immutable contract, signs the transaction and sends it
// checks contract has been created correctly
func createContract(t *testing.T, baseApp weave_app.BaseApp, chainID string, height int64, signers []Signer,
	activationThreshold int64, contractSigs ...[]byte) []byte {
	msg := &multisig.CreateContractMsg{
		Sigs:                contractSigs,
		ActivationThreshold: activationThreshold,
		AdminThreshold:      int64(len(contractSigs)) + 1, // immutable
	}

	tx := &app.Tx{
		Sum: &app.Sum{&app.Sum_CreateContractMsg{msg}},
	}

	dres := signAndCommit(t, baseApp, tx, signers, chainID, height)

	// retrieve contract ID and check contract was correctly created
	contractID := dres.Data
	queryAndCheckContract(t, baseApp, "/contracts", contractID,
		multisig.Contract{
			Sigs:                contractSigs,
			ActivationThreshold: activationThreshold,
			AdminThreshold:      int64(len(contractSigs)) + 1,
		})

	return contractID
}

// signAndCommit signs tx with signatures from signers and submits to the chain
// asserts and fails the test in case of errors during the process
func signAndCommit(t *testing.T, app weave_app.BaseApp, tx *app.Tx, signers []Signer, chainID string,
	height int64) abci.ResponseDeliverTx {
	for _, signer := range signers {
		sig, err := sigs.SignTx(signer.pk, tx, chainID, signer.nonce)
		require.NoError(t, err)
		tx.Signatures = append(tx.Signatures, sig)
	}

	txBytes, err := tx.Marshal()
	require.NoError(t, err)
	require.NotEmpty(t, txBytes)

	// Submit to the chain
	header := abci.Header{Height: height}
	app.BeginBlock(abci.RequestBeginBlock{Header: header})
	// check and deliver must pass
	chres := app.CheckTx(txBytes)
	require.Equal(t, uint32(0), chres.Code, chres.Log)

	dres := app.DeliverTx(txBytes)
	require.Equal(t, uint32(0), dres.Code, dres.Log)

	app.EndBlock(abci.RequestEndBlock{})
	cres := app.Commit()
	assert.NotEmpty(t, cres.Data)
	return dres
}

// queryAndCheckWallet queries the wallet from the chain and check it is the one expected
func queryAndCheckWallet(t *testing.T, baseApp weave_app.BaseApp, path string, data []byte, expected namecoin.Wallet) {
	query := abci.RequestQuery{Path: path, Data: data}
	res := baseApp.Query(query)

	// check query was ok
	require.Equal(t, uint32(0), res.Code, "%#v", res)
	assert.NotEmpty(t, res.Value)

	var actual namecoin.Wallet
	err := weave_app.UnmarshalOneResult(res.Value, &actual)
	require.NoError(t, err)
	require.Equal(t, expected, actual)
}

// queryAndCheckContract queries the contract from the chain and check it is the one expected
func queryAndCheckContract(t *testing.T, baseApp weave_app.BaseApp, path string, data []byte, expected multisig.Contract) {
	query := abci.RequestQuery{Path: path, Data: data}
	res := baseApp.Query(query)

	// check query was ok
	require.Equal(t, uint32(0), res.Code, "%#v", res)
	assert.NotEmpty(t, res.Value)

	var actual multisig.Contract
	err := weave_app.UnmarshalOneResult(res.Value, &actual)
	require.NoError(t, err)
	require.Equal(t, expected, actual)
}

// queryAndCheckToken queries tokens from the chain and check they're the one expected
func queryAndCheckToken(t *testing.T, baseApp weave_app.BaseApp, path string, data []byte, expected []namecoin.Token) {
	query := abci.RequestQuery{Path: path, Data: data}
	res := baseApp.Query(query)

	var set weave_app.ResultSet
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
