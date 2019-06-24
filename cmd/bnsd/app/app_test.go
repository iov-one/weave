package bnsd_test

import (
	"encoding/hex"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/iov-one/weave"
	weaveApp "github.com/iov-one/weave/app"
	"github.com/iov-one/weave/cmd/bnsd/app"
	"github.com/iov-one/weave/cmd/bnsd/app/testdata/fixtures"
	"github.com/iov-one/weave/coin"
	"github.com/iov-one/weave/crypto"
	"github.com/iov-one/weave/x/batch"
	"github.com/iov-one/weave/x/cash"
	"github.com/iov-one/weave/x/multisig"
	"github.com/iov-one/weave/x/sigs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/libs/common"
)

func TestApp(t *testing.T) {
	appFixture := fixtures.NewApp()
	addr := appFixture.GenesisKeyAddress
	pk := appFixture.GenesisKey
	chainID := appFixture.ChainID
	myApp := appFixture.Build()
	// Query for my balance
	dbKey := cash.NewBucket().DBKey(appFixture.GenesisKeyAddress)

	queryAndCheckAccount(t, myApp, "/", dbKey, cash.Set{
		Metadata: &weave.Metadata{Schema: 1},
		Coins: coin.Coins{
			{Ticker: "ETH", Whole: 50000},
			{Ticker: "FRNK", Whole: 1234},
		},
	})

	// build and sign a transaction
	pk2 := crypto.GenPrivKeyEd25519()
	addr2 := pk2.PublicKey().Address()
	dres := sendToken(t, myApp, appFixture.ChainID, 2, []Signer{{pk, 0}}, addr, addr2, 2000, "ETH", "Have a great trip!")

	// ensure 4 keys for all accounts that are modified by a transaction
	require.Equal(t, 4, len(dres.Tags), tagsAsString(dres.Tags))
	feeDistAddr := weave.NewCondition("dist", "revenue", []byte{0, 0, 0, 0, 0, 0, 0, 1}).Address()
	wantKeys := []string{
		toHex("cash:") + addr.String(),        // sender balance decreased
		toHex("cash:") + addr2.String(),       // receiver balance increased
		toHex("sigs:") + addr.String(),        // sender sequence incremented
		toHex("cash:") + feeDistAddr.String(), // fee destination
	}
	for _, want := range wantKeys {
		var found bool
		for i := 0; i < len(dres.Tags) && !found; i++ {
			found = string(dres.Tags[i].Key) == want
		}
		require.True(t, found, "not found tag %s in %s", want, tagsAsString(dres.Tags))
	}

	require.Equal(t, []string{"s", "s", "s", "s"}, []string{
		string(dres.Tags[0].Value),
		string(dres.Tags[1].Value),
		string(dres.Tags[2].Value),
		string(dres.Tags[3].Value),
	})

	// Query for fees stored
	queryAndCheckAccount(t, myApp, "/wallets", feeDistAddr, cash.Set{
		Coins: coin.Coins{
			{Ticker: "FRNK", Whole: 1},
		},
	})
	// Query for new balances (same query, new state)
	queryAndCheckAccount(t, myApp, "/", dbKey, cash.Set{
		Metadata: &weave.Metadata{Schema: 1},
		Coins: coin.Coins{
			{Ticker: "ETH", Whole: 48000},
			{Ticker: "FRNK", Whole: 1233},
		},
	})

	// make sure money arrived safely
	dbKeyReceiver := cash.NewBucket().DBKey(addr2)
	queryAndCheckAccount(t, myApp, "/", dbKeyReceiver, cash.Set{
		Metadata: &weave.Metadata{Schema: 1},
		Coins: coin.Coins{
			{
				Ticker: "ETH",
				Whole:  2000,
			},
		},
	})

	// make sure other paths also get this value....
	queryAndCheckAccount(t, myApp, "/wallets", addr2, cash.Set{
		Metadata: &weave.Metadata{Schema: 1},
		Coins:    coin.Coins{{Ticker: "ETH", Whole: 2000}},
	})

	// make sure other paths also get this value....
	queryAndCheckAccount(t, myApp, "/wallets?prefix", addr2[:15], cash.Set{
		Metadata: &weave.Metadata{Schema: 1},
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
	sendToken(t, myApp, chainID, 6, []Signer{{pk, 4}},
		addr, safeKeyContractAddr, 10, "FRNK", "Fees - New wallet controlled by safeKeyContract")

	// build and sign a transaction using master key to activate safeKeyContract
	receiver := crypto.GenPrivKeyEd25519()
	sendToken(t, myApp, chainID, 7, []Signer{{masterKey, 0}},
		safeKeyContractAddr, receiver.PublicKey().Address(), 1000, "ETH", "Gift from a contract!", safeKeyContract)
	// verify money was received
	queryAndCheckAccount(t, myApp, "/wallets", receiver.PublicKey().Address(), cash.Set{
		Metadata: &weave.Metadata{Schema: 1},
		Coins:    coin.Coins{{Ticker: "ETH", Whole: 1000}},
	})

	// Now do the same operation but using recoveryContract to activate safeKeyContract
	// create a new receiver so it is easy to check its balance (no need to remember previous one)
	receiver = crypto.GenPrivKeyEd25519()
	sendToken(t, myApp, chainID, 8, []Signer{{recovery1, 0}, {recovery2, 0}},
		safeKeyContractAddr, receiver.PublicKey().Address(), 1000, "ETH", "Another gift from a contract!",
		recoveryContract, safeKeyContract)
	// verify money was received
	queryAndCheckAccount(t, myApp, "/wallets", receiver.PublicKey().Address(), cash.Set{
		Metadata: &weave.Metadata{Schema: 1},
		Coins:    coin.Coins{{Ticker: "ETH", Whole: 1000}},
	})

	// Now we send a batch request to a new recipient
	batchReceiver := crypto.GenPrivKeyEd25519()
	batchAddr := batchReceiver.PublicKey().Address()
	sendBatch(t, myApp, chainID, 9, []Signer{{pk, 5}}, addr, batchAddr, 20, "ETH", "And the cash keeps flowing")
}

func tagsAsString(pairs []common.KVPair) string {
	r := make([]string, len(pairs))
	for i, v := range pairs {
		x, err := hex.DecodeString(string(v.Key))
		if err != nil {
			panic(err)
		}
		// decode prefix: 5 prefix in this scenarios
		r[i] = string(x[0:5]) + string(v.Key[hex.EncodedLen(5):])
	}
	return strings.Join(r, ";")
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
func sendToken(t *testing.T, baseApp abci.Application, chainID string, height int64, signers []Signer,
	from, to []byte, amount int64, ticker, memo string, contracts ...[]byte) abci.ResponseDeliverTx {
	msg := &cash.SendMsg{
		Metadata: &weave.Metadata{Schema: 1},
		Src:      from,
		Dest:     to,
		Amount:   &coin.Coin{Whole: amount, Ticker: ticker},
		Memo:     memo,
	}
	tx := &app.Tx{
		Sum:      &app.Tx_SendMsg{SendMsg: msg},
		Multisig: contracts,
	}
	tx.Fee(from, coin.NewCoin(1, 0, "FRNK"))
	res := signAndCommit(t, baseApp, tx, signers, chainID, height)
	return res
}

// checks batch works
func sendBatch(t *testing.T, baseApp abci.Application, chainID string, height int64, signers []Signer,
	from, to weave.Address, amount int64, ticker, memo string, contracts ...[]byte) {
	msg := &cash.SendMsg{
		Metadata: &weave.Metadata{Schema: 1},
		Src:      from,
		Dest:     to,
		Amount: &coin.Coin{
			Whole:  amount,
			Ticker: ticker,
		},
		Memo: memo,
	}

	var messages []app.ExecuteBatchMsg_Union
	for i := 0; i < batch.MaxBatchMessages; i++ {
		messages = append(messages,
			app.ExecuteBatchMsg_Union{
				Sum: &app.ExecuteBatchMsg_Union_SendMsg{
					SendMsg: msg,
				},
			})
	}

	tx := &app.Tx{
		Sum: &app.Tx_ExecuteBatchMsg{
			ExecuteBatchMsg: &app.ExecuteBatchMsg{
				Messages: messages,
			},
		},
	}
	tx.Fee(from, coin.NewCoin(1, 0, "FRNK"))

	dres := signAndCommit(t, baseApp, tx, signers, chainID, height)

	// make sure the tags are only present once (not once per item)
	feeDistAddr := weave.NewCondition("dist", "revenue", []byte{0, 0, 0, 0, 0, 0, 0, 1}).Address()
	if len(dres.Tags) != 4 {
		t.Fatalf("%#v", dres.Tags)
	}
	wantKeys := []string{
		toHex("cash:") + from.String(),
		toHex("cash:") + to.String(),
		toHex("sigs:") + from.String(),
		toHex("cash:") + feeDistAddr.String(), // fee destination
	}
	sort.Strings(wantKeys)
	gotKeys := []string{
		string(dres.Tags[0].Key),
		string(dres.Tags[1].Key),
		string(dres.Tags[2].Key),
		string(dres.Tags[3].Key),
	}
	assert.Equal(t, wantKeys, gotKeys)

	checkAmount := amount * batch.MaxBatchMessages

	// make sure money arrived only for successful batch
	queryAndCheckAccount(t, baseApp, "/wallets", to, cash.Set{
		Metadata: &weave.Metadata{Schema: 1},
		Coins: coin.Coins{
			{Ticker: ticker, Whole: checkAmount},
		},
	})
}

// createContract creates an immutable contract, signs the transaction and sends it
// checks contract has been created correctly
func createContract(
	t *testing.T,
	baseApp abci.Application,
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
			Weight:    1,
		}
	}
	msg := &multisig.CreateContractMsg{
		Metadata:            &weave.Metadata{Schema: 1},
		Participants:        participants,
		ActivationThreshold: activationThreshold,
		AdminThreshold:      multisig.Weight(len(contractSigs)) + 1, // immutable
	}

	tx := &app.Tx{
		Sum: &app.Tx_CreateContractMsg{CreateContractMsg: msg},
	}

	tx.Fee(signers[0].pk.PublicKey().Address(), coin.NewCoin(1, 0, "FRNK"))
	dres := signAndCommit(t, baseApp, tx, signers, chainID, height)

	// retrieve contract ID and check contract was correctly created
	contractID := dres.Data
	queryAndCheckContract(t, baseApp, "/contracts", contractID,
		multisig.Contract{
			Metadata:            &weave.Metadata{Schema: 1},
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
	app abci.Application,
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
	header := abci.Header{
		Height: height,
		Time:   time.Now(),
	}
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
func queryAndCheckAccount(t *testing.T, baseApp abci.Application, path string, data []byte, expected cash.Set) {
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
func queryAndCheckContract(t *testing.T, baseApp abci.Application, path string, data []byte, expected multisig.Contract) {
	query := abci.RequestQuery{Path: path, Data: data}
	res := baseApp.Query(query)

	// check query was ok
	require.Equal(t, uint32(0), res.Code, "%#v", res)
	assert.NotEmpty(t, res.Value)

	actual := multisig.Contract{
		Metadata: &weave.Metadata{Schema: 1},
	}
	err := weaveApp.UnmarshalOneResult(res.Value, &actual)
	require.NoError(t, err)
	require.Equal(t, expected, actual)
}
