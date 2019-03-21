package app_test

import (
	"encoding/hex"
	"sort"
	"strings"
	"testing"

	weaveApp "github.com/iov-one/weave/app"
	"github.com/iov-one/weave/cmd/bnsd/app"
	"github.com/iov-one/weave/cmd/bnsd/app/testdata/fixtures"
	"github.com/iov-one/weave/coin"
	"github.com/iov-one/weave/crypto"
	"github.com/iov-one/weave/x/cash"
	"github.com/iov-one/weave/x/multisig"
	"github.com/iov-one/weave/x/sigs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	abci "github.com/tendermint/tendermint/abci/types"
)

func TestApp(t *testing.T) {
	appFixture := fixtures.NewApp()
	addr := appFixture.GenesisKeyAddress
	pk := appFixture.GenesisKey
	chainID := appFixture.ChainID
	myApp := appFixture.Build()
	// Query for my balance
	key := cash.NewBucket().DBKey(appFixture.GenesisKeyAddress)
	queryAndCheckAccount(t, myApp, "/", key, cash.Set{
		Coins: coin.Coins{
			{Ticker: "ETH", Whole: 50000},
			{Ticker: "FRNK", Whole: 1234},
		},
	})

	// build and sign a transaction
	pk2 := crypto.GenPrivKeyEd25519()
	addr2 := pk2.PublicKey().Address()
	dres := sendToken(t, myApp, appFixture.ChainID, 2, []Signer{{pk, 0}}, addr, addr2, 2000, "ETH", "Have a great trip!")

	// ensure 3 keys with proper values
	if assert.Equal(t, 3, len(dres.Tags), "%#v", dres.Tags) {
		wantKeys := []string{
			toHex("cash:") + addr.String(),
			toHex("cash:") + addr2.String(),
			toHex("sigs:") + addr.String(),
		}
		sort.Strings(wantKeys)
		gotKeys := []string{
			string(dres.Tags[0].Key),
			string(dres.Tags[1].Key),
			string(dres.Tags[2].Key),
		}
		assert.Equal(t, wantKeys, gotKeys)

		assert.Equal(t, []string{"s", "s", "s"}, []string{
			string(dres.Tags[0].Value),
			string(dres.Tags[1].Value),
			string(dres.Tags[2].Value),
		})
	}

	// Query for new balances (same query, new state)
	queryAndCheckAccount(t, myApp, "/", key, cash.Set{
		Coins: coin.Coins{
			{Ticker: "ETH", Whole: 48000},
			{Ticker: "FRNK", Whole: 1234},
		},
	})

	// make sure money arrived safely
	key2 := cash.NewBucket().DBKey(addr2)
	queryAndCheckAccount(t, myApp, "/", key2, cash.Set{
		Coins: coin.Coins{
			{
				Ticker: "ETH",
				Whole:  2000,
			},
		},
	})

	// make sure other paths also get this value....
	queryAndCheckAccount(t, myApp, "/wallets", addr2, cash.Set{
		Coins: coin.Coins{{Ticker: "ETH", Whole: 2000}},
	})

	// make sure other paths also get this value....
	queryAndCheckAccount(t, myApp, "/wallets?prefix", addr2[:15], cash.Set{
		Coins: coin.Coins{
			{Ticker: "ETH", Whole: 2000},
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

func toHex(s string) string {
	h := hex.EncodeToString([]byte(s))
	return strings.ToUpper(h)
}

type Signer struct {
	pk    *crypto.PrivateKey
	nonce int64
}

// sendToken creates the transaction, signs it and sends it
// checks money has arrived safely
func sendToken(t *testing.T, baseApp weaveApp.BaseApp, chainID string, height int64, signers []Signer,
	from, to []byte, amount int64, ticker, memo string, contracts ...[]byte) abci.ResponseDeliverTx {
	msg := &cash.SendMsg{
		Src:    from,
		Dest:   to,
		Amount: &coin.Coin{Whole: amount, Ticker: ticker},
		Memo:   memo,
	}
	tx := &app.Tx{
		Sum:      &app.Tx_SendMsg{SendMsg: msg},
		Multisig: contracts,
	}
	res := signAndCommit(t, baseApp, tx, signers, chainID, height)
	// make sure money arrived safely
	queryAndCheckAccount(t, baseApp, "/wallets", to, cash.Set{Coins: coin.Coins{{Ticker: ticker, Whole: amount}}})
	return res
}

// createContract creates an immutable contract, signs the transaction and sends it
// checks contract has been created correctly
func createContract(
	t *testing.T,
	baseApp weaveApp.BaseApp,
	chainID string,
	height int64,
	signers []Signer,
	activationThreshold multisig.Weight,
	contractSigs ...[]byte,
) []byte {
	participants := make([]*multisig.Participant, len(contractSigs))
	for i, addr := range contractSigs {
		participants[i] = &multisig.Participant{
			Signature: addr,
			Power:     1,
		}
	}
	msg := &multisig.CreateContractMsg{
		Participants:        participants,
		ActivationThreshold: activationThreshold,
		AdminThreshold:      multisig.Weight(len(contractSigs)) + 1, // immutable
	}

	tx := &app.Tx{
		Sum: &app.Tx_CreateContractMsg{CreateContractMsg: msg},
	}

	dres := signAndCommit(t, baseApp, tx, signers, chainID, height)

	// retrieve contract ID and check contract was correctly created
	contractID := dres.Data
	queryAndCheckContract(t, baseApp, "/contracts", contractID,
		multisig.Contract{
			Participants:        participants,
			ActivationThreshold: activationThreshold,
			AdminThreshold:      multisig.Weight(len(contractSigs)) + 1,
		})

	return contractID
}

// signAndCommit signs tx with signatures from signers and submits to the chain
// asserts and fails the test in case of errors during the process
func signAndCommit(
	t *testing.T,
	app weaveApp.BaseApp,
	tx *app.Tx,
	signers []Signer,
	chainID string,
	height int64,
) abci.ResponseDeliverTx {
	t.Helper()

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

// queryAndCheckAccount queries the wallet from the chain and check it is the one expected
func queryAndCheckAccount(t *testing.T, baseApp weaveApp.BaseApp, path string, data []byte, expected cash.Set) {
	t.Helper()

	query := abci.RequestQuery{Path: path, Data: data}
	res := baseApp.Query(query)

	// check query was ok
	require.Equal(t, uint32(0), res.Code, "%#v", res)
	assert.NotEmpty(t, res.Value)

	var actual cash.Set
	err := weaveApp.UnmarshalOneResult(res.Value, &actual)
	require.NoError(t, err)
	require.Equal(t, expected.Coins, actual.Coins)
}

// queryAndCheckContract queries the contract from the chain and check it is the one expected
func queryAndCheckContract(t *testing.T, baseApp weaveApp.BaseApp, path string, data []byte, expected multisig.Contract) {
	query := abci.RequestQuery{Path: path, Data: data}
	res := baseApp.Query(query)

	// check query was ok
	require.Equal(t, uint32(0), res.Code, "%#v", res)
	assert.NotEmpty(t, res.Value)

	var actual multisig.Contract
	err := weaveApp.UnmarshalOneResult(res.Value, &actual)
	require.NoError(t, err)
	require.Equal(t, expected, actual)
}
