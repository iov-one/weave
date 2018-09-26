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

	cmn "github.com/tendermint/tmlibs/common"
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
	dres := sendToken(t, myApp, chainID, 2, []Signer{{pk, 0}}, addr, addr2, 2000, "ETH", "Have a great trip!")

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
func sendToken(t require.TestingT, baseApp app.BaseApp, chainID string, height int64, signers []Signer,
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

	tx := &Tx{
		Sum:      &Tx_SendMsg{msg},
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
func createContract(t require.TestingT, baseApp app.BaseApp, chainID string, height int64, signers []Signer,
	activationThreshold int64, contractSigs ...[]byte) []byte {
	msg := &multisig.CreateContractMsg{
		Sigs:                contractSigs,
		ActivationThreshold: activationThreshold,
		AdminThreshold:      int64(len(contractSigs)) + 1, // immutable
	}

	tx := &Tx{
		Sum: &Tx_CreateContractMsg{msg},
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
func signAndCommit(t require.TestingT, app app.BaseApp, tx *Tx, signers []Signer, chainID string,
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
func queryAndCheckWallet(t require.TestingT, baseApp app.BaseApp, path string, data []byte, expected namecoin.Wallet) {
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

// queryAndCheckContract queries the contract from the chain and check it is the one expected
func queryAndCheckContract(t require.TestingT, baseApp app.BaseApp, path string, data []byte, expected multisig.Contract) {
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

// queryAndCheckToken queries tokens from the chain and check they're the one expected
func queryAndCheckToken(t require.TestingT, baseApp app.BaseApp, path string, data []byte, expected []namecoin.Token) {
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

type account struct {
	pk *crypto.PrivateKey
	n  int64
}

func (a *account) nonce() (n int64) {
	n = a.n
	a.n++
	return
}

func (a *account) address() []byte {
	return a.pk.PublicKey().Address()
}

func newBenchmarkApp(t require.TestingT, chainID string, accounts []*account) app.BaseApp {
	// no minimum fee, in-memory data-store
	abciApp, err := GenerateApp("", log.NewNopLogger(), true)
	require.NoError(t, err)
	myApp := abciApp.(app.BaseApp) // let's set up a genesis file with some cash
	pk := crypto.GenPrivKeyEd25519()
	addr := pk.PublicKey().Address()
	appState := fmt.Sprintf(`{
        "wallets": [{
            "name": "demote",
            "address": "%s",
            "coins": [{
                "whole": 1234567890,
                "ticker": "IOV"
            }]
		}],
        "tokens": [{
            "ticker": "IOV",
            "name": "Smells like ethereum",
            "sig_figs": 9
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

	for idx, acc := range accounts {
		sendToken(t, myApp, chainID, int64(idx+2), []Signer{{pk, int64(idx)}}, pk.PublicKey().Address(), acc.address(), 1000, "IOV", "benchmark")
	}

	return myApp
}

func makeTx(t require.TestingT, chainID string, sender, receiver *account) []byte {
	msg := &cash.SendMsg{
		Src:  sender.address(),
		Dest: receiver.address(),
		Amount: &x.Coin{
			Whole:  1,
			Ticker: "IOV",
		},
		Memo: "courtesy of benchmark",
	}

	tx := &Tx{
		Sum: &Tx_SendMsg{msg},
	}

	nonce := sender.nonce()
	sig, err := sigs.SignTx(sender.pk, tx, chainID, nonce)
	require.NoError(t, err)
	tx.Signatures = append(tx.Signatures, sig)
	txBytes, err := tx.Marshal()
	require.NoError(t, err)
	require.NotEmpty(t, txBytes)
	return txBytes
}

func SendTxBenchRunner(b *testing.B, nbAccounts, blockSize int) {
	accounts := make([]*account, nbAccounts)
	for i := 0; i < nbAccounts; i++ {
		accounts[i] = &account{pk: crypto.GenPrivKeyEd25519()}
	}

	chainID := "bench-net-22"
	myApp := newBenchmarkApp(b, chainID, accounts)

	txs := make([][]byte, b.N)
	for i := 1; i <= b.N; i++ {
		sender := accounts[cmn.RandInt()%nbAccounts]
		recipient := accounts[cmn.RandInt()%nbAccounts]
		txs[i-1] = makeTx(b, chainID, sender, recipient)
	}

	b.ResetTimer()

	for i := 1; i <= b.N; i++ {
		chres := myApp.CheckTx(txs[i-1])
		require.Equal(b, uint32(0), chres.Code, chres.Log)
	}

	k := 1
	header := abci.Header{Height: int64(k)}
	myApp.BeginBlock(abci.RequestBeginBlock{Header: header})
	for i := 1; i <= b.N; i++ {
		if i%blockSize == 0 {
			myApp.EndBlock(abci.RequestEndBlock{})
			cres := myApp.Commit()
			assert.NotEmpty(b, cres.Data)

			k++
			header = abci.Header{Height: int64(k)}
			myApp.BeginBlock(abci.RequestBeginBlock{Header: header})
		}

		dres := myApp.DeliverTx(txs[i-1])
		require.Equal(b, uint32(0), dres.Code, dres.Log)
	}

	myApp.EndBlock(abci.RequestEndBlock{})
	cres := myApp.Commit()
	assert.NotEmpty(b, cres.Data)
}
func TestNewBenchmarkApp(t *testing.T) {
	tests := []struct {
		accounts  int
		blockSize int
	}{
		{100, 10},
		{100, 200},
		{10000, 1000},
		{10000, 2000},
	}

	for _, tt := range tests {
		prefix := fmt.Sprintf("%d-%d", tt.accounts, tt.blockSize)
		t.Run(prefix, func(sub *testing.T) {
			accounts := make([]*account, tt.accounts)
			for i := 0; i < tt.accounts; i++ {
				accounts[i] = &account{pk: crypto.GenPrivKeyEd25519()}
			}

			chainID := prefix
			newBenchmarkApp(t, chainID, accounts)
		})
	}
}

func BenchmarkSendTx(b *testing.B) {
	benchmarks := []struct {
		accounts  int
		blockSize int
	}{
		{100, 10},
		{100, 200},
		{10000, 1000},
		{10000, 2000},
	}

	for _, bb := range benchmarks {
		prefix := fmt.Sprintf("%d-%d", bb.accounts, bb.blockSize)
		b.Run(prefix, func(sub *testing.B) {
			SendTxBenchRunner(sub, bb.accounts, bb.blockSize)
		})
	}
}
