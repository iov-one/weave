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

func TestSendTx(t *testing.T) {
	chainID := "test-net-22"
	mainAccount := &account{pk: crypto.GenPrivKeyEd25519()}
	myApp := newTestApp(t, chainID, []*account{mainAccount})

	// Query for my balance
	key := namecoin.NewWalletBucket().DBKey(mainAccount.address())
	queryAndCheckWallet(t, myApp, "/", key,
		namecoin.Wallet{
			Name: "wallet0",
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
	dres := sendToken(t, myApp, chainID, 2, []*account{mainAccount}, mainAccount.address(), addr2, 2000, "ETH", "Have a great trip!")

	// ensure 3 keys with proper values
	if assert.Equal(t, 3, len(dres.Tags), "%#v", dres.Tags) {
		// three keys we expect, in order
		keys := make([][]byte, 3)
		vals := [][]byte{[]byte("s"), []byte("s"), []byte("s")}
		hexWllt := []byte("776C6C743A")
		hexSigs := []byte("736967733A")
		addr := mainAccount.pk.PublicKey().Address()
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
			Name: "wallet0",
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
}

func TestQuery(t *testing.T) {
	chainID := "test-net-22"
	mainAccount := &account{pk: crypto.GenPrivKeyEd25519()}
	myApp := newTestApp(t, chainID, []*account{mainAccount})

	// build and sign a transaction
	pk2 := crypto.GenPrivKeyEd25519()
	addr2 := pk2.PublicKey().Address()
	sendToken(t, myApp, chainID, 2, []*account{mainAccount}, mainAccount.address(), addr2, 2000, "ETH", "Have a great trip!")

	// Query for new balances
	key := namecoin.NewWalletBucket().DBKey(mainAccount.address())
	queryAndCheckWallet(t, myApp, "/", key,
		namecoin.Wallet{
			Name: "wallet0",
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
	queryAndCheckWallet(t, myApp, "/wallets/name", []byte("wallet0"),
		namecoin.Wallet{
			Name: "wallet0",
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
}

func TestMultisigContract(t *testing.T) {
	chainID := "test-net-22"
	mainAccount := &account{pk: crypto.GenPrivKeyEd25519()}
	myApp := newTestApp(t, chainID, []*account{mainAccount})

	// create recoveryContract
	recovery1 := crypto.GenPrivKeyEd25519()
	recovery2 := crypto.GenPrivKeyEd25519()
	recovery3 := crypto.GenPrivKeyEd25519()
	recoveryContract := createContract(t, myApp, chainID, 3, []*account{mainAccount},
		2, recovery1.PublicKey().Address(), recovery2.PublicKey().Address(), recovery3.PublicKey().Address())

	// create safeKeyContract contract
	// can be activated by masterKey or recoveryContract
	masterKey := crypto.GenPrivKeyEd25519()
	safeKeyContract := createContract(t, myApp, chainID, 4, []*account{mainAccount},
		1, masterKey.PublicKey().Address(), multisig.MultiSigCondition(recoveryContract).Address())

	// create a wallet controlled by safeKeyContract
	safeKeyContractAddr := multisig.MultiSigCondition(safeKeyContract).Address()
	sendToken(t, myApp, chainID, 5, []*account{mainAccount},
		mainAccount.address(), safeKeyContractAddr, 2000, "ETH", "New wallet controlled by safeKeyContract")

	// build and sign a transaction using master key to activate safeKeyContract
	receiver := crypto.GenPrivKeyEd25519()
	sendToken(t, myApp, chainID, 6, []*account{{pk: masterKey}},
		safeKeyContractAddr, receiver.PublicKey().Address(), 1000, "ETH", "Gift from a contract!", safeKeyContract)

	// Now do the same operation but using recoveryContract to activate safeKeyContract
	// create a new receiver so it is easy to check its balance (no need to remember previous one)
	receiver = crypto.GenPrivKeyEd25519()
	sendToken(t, myApp, chainID, 7, []*account{{pk: recovery1}, {pk: recovery2}},
		safeKeyContractAddr, receiver.PublicKey().Address(), 1000, "ETH", "Another gift from a contract!",
		recoveryContract, safeKeyContract)
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

// newTestApp creates a new app with a wallet for each account
// coins and tokens are the same across all accounts and calls
func newTestApp(t require.TestingT, chainID string, accounts []*account) app.BaseApp {
	// no minimum fee, in-memory data-store
	abciApp, err := GenerateApp("", log.NewNopLogger(), true)
	require.NoError(t, err)
	myApp := abciApp.(app.BaseApp) // let's set up a genesis file with some cash

	var wBuffer bytes.Buffer
	for i, acc := range accounts {
		_, err := wBuffer.WriteString(fmt.Sprintf(`{
            "name": "wallet%d",
			"address": "%s",
			"coins": [{
                "whole": 50000,
                "ticker": "ETH"
            },{
                "whole": 1234,
				"ticker": "FRNK"
			}]
		}`, i, acc.pk.PublicKey().Address()))
		require.NoError(t, err)

		if i != len(accounts)-1 {
			_, err = wBuffer.WriteString(",")
			require.NoError(t, err)
		}
	}

	appState := fmt.Sprintf(`{
        "wallets": [%s],
        "tokens": [{
            "ticker": "ETH",
            "name": "Smells like ethereum",
            "sig_figs": 9
        },{
            "ticker": "FRNK",
            "name": "Frankie",
            "sig_figs": 3
		}]
	}`, wBuffer.String())

	// Commit first block, make sure non-nil hash
	myApp.InitChain(abci.RequestInitChain{AppStateBytes: []byte(appState), ChainId: chainID})
	header := abci.Header{Height: 1}
	myApp.BeginBlock(abci.RequestBeginBlock{Header: header})
	myApp.EndBlock(abci.RequestEndBlock{})
	cres := myApp.Commit()
	block1 := cres.Data
	assert.NotEmpty(t, block1)
	assert.Equal(t, chainID, myApp.GetChainID())

	return myApp
}

// sendToken creates the transaction, signs it and sends it
// checks money has arrived safely
func sendToken(t require.TestingT, baseApp app.BaseApp, chainID string, height int64, signers []*account,
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
func createContract(t require.TestingT, baseApp app.BaseApp, chainID string, height int64, signers []*account,
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
func signAndCommit(t require.TestingT, app app.BaseApp, tx *Tx, signers []*account, chainID string,
	height int64) abci.ResponseDeliverTx {
	for _, signer := range signers {
		sig, err := sigs.SignTx(signer.pk, tx, chainID, signer.nonce())
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

// makeSendTx is a special case of sendToken when the sender account is the only signer
// this is used in our benchmark
func makeSendTx(t require.TestingT, chainID string, sender, receiver *account, ticker, memo string, amount int64, contracts ...[]byte) []byte {
	msg := &cash.SendMsg{
		Src:  sender.address(),
		Dest: receiver.address(),
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

	sig, err := sigs.SignTx(sender.pk, tx, chainID, sender.nonce())
	require.NoError(t, err)
	tx.Signatures = append(tx.Signatures, sig)
	txBytes, err := tx.Marshal()
	require.NoError(t, err)
	require.NotEmpty(t, txBytes)
	return txBytes
}

func makeCreateContractTx(t require.TestingT, chainID string, signers [][]byte, threshold int64) *Tx {
	msg := &multisig.CreateContractMsg{
		Sigs:                signers,
		ActivationThreshold: threshold,
		AdminThreshold:      threshold,
	}

	return &Tx{
		Sum: &Tx_CreateContractMsg{msg},
	}
}

func newBlock(t require.TestingT, baseApp app.BaseApp, txs [][]byte, nbAccounts, blockSize, maxBlock, idx, height int) {
	idTx := func(a, b int) int { return (a * blockSize) + b }
	for k := 0; k < blockSize && idTx(idx, k) < maxBlock; k++ {
		chres := baseApp.CheckTx(txs[idTx(idx, k)])
		require.Equal(t, uint32(0), chres.Code, chres.Log)
	}

	header := abci.Header{Height: int64(height)}
	baseApp.BeginBlock(abci.RequestBeginBlock{Header: header})

	for k := 0; k < blockSize && idTx(idx, k) < maxBlock; k++ {
		dres := baseApp.DeliverTx(txs[idTx(idx, k)])
		require.Equal(t, uint32(0), dres.Code, dres.Log)
	}

	baseApp.EndBlock(abci.RequestEndBlock{})
	cres := baseApp.Commit()
	assert.NotEmpty(t, cres.Data)
}

// benchmarkSendTx runs the actual benchmark sequence eg.
// N * CheckTx
// BeginBlock
// N * DeliverTx
// EndBlock
// Commit
func benchmarkSendTxWithMultisig(b *testing.B, nbAccounts, blockSize, nbMultisigSigs int, threshold int64) {
	accounts := make([]*account, nbAccounts)
	for i := 0; i < nbAccounts; i++ {
		accounts[i] = &account{pk: crypto.GenPrivKeyEd25519()}
	}

	chainID := "bench-net-22"
	myApp := newTestApp(b, chainID, accounts)

	height := 1
	txs := make([][]byte, b.N)
	for i := 0; i < b.N; i++ {
		multisigs := make([][]byte, nbMultisigSigs)
		for k := 0; k < nbMultisigSigs; k++ {
			multisigs[k] = accounts[cmn.RandInt()%nbAccounts].address()
		}

		tx := makeCreateContractTx(b, chainID, multisigs, threshold)
		res := signAndCommit(b, myApp, tx, []*account{accounts[cmn.RandInt()%nbAccounts]}, chainID, int64(height))
		height++

		sender := accounts[cmn.RandInt()%nbAccounts]
		recipient := accounts[cmn.RandInt()%nbAccounts]
		txs[i] = makeSendTx(b, chainID, sender, recipient, "ETH", "benchmark", 1, res.Data)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		newBlock(b, myApp, txs, nbAccounts, blockSize, b.N, i, i+height)
	}
}

// benchmarkSendTx runs the actual benchmark sequence eg.
// N * CheckTx
// BeginBlock
// N * DeliverTx
// EndBlock
// Commit
func benchmarkSendTx(b *testing.B, nbAccounts, blockSize int) {
	accounts := make([]*account, nbAccounts)
	for i := 0; i < nbAccounts; i++ {
		accounts[i] = &account{pk: crypto.GenPrivKeyEd25519()}
	}

	chainID := "bench-net-22"
	myApp := newTestApp(b, chainID, accounts)

	txs := make([][]byte, b.N)
	for i := 0; i < b.N; i++ {
		sender := accounts[cmn.RandInt()%nbAccounts]
		recipient := accounts[cmn.RandInt()%nbAccounts]
		txs[i] = makeSendTx(b, chainID, sender, recipient, "ETH", "benchmark", 1)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		newBlock(b, myApp, txs, nbAccounts, blockSize, b.N, i, i+1)
	}
}

// Runs benchmarks with various input combination of initial accounts and block size
func BenchmarkSendTx(b *testing.B) {
	benchmarks := []struct {
		accounts  int
		blockSize int
	}{
		{100, 10},
		{100, 100},
		{10000, 10},
		{10000, 100},
		{10000, 1000},
		{100000, 10},
		{100000, 100},
		{100000, 1000},
	}

	for _, bb := range benchmarks {
		prefix := fmt.Sprintf("%d-%d", bb.accounts, bb.blockSize)
		b.Run(prefix, func(sub *testing.B) {
			benchmarkSendTx(sub, bb.accounts, bb.blockSize)
		})
	}
}

func BenchmarkSendTxMultiSig(b *testing.B) {
	benchmarks := []struct {
		accounts  int
		blockSize int
		nbSigs    int
		threshold int64
	}{
		{10000, 10, 2, 1},
		{10000, 10, 3, 2},
		{10000, 100, 2, 1},
		{10000, 100, 3, 2},
		{10000, 1000, 2, 1},
		{10000, 1000, 3, 2},
		{100000, 100, 2, 1},
		{100000, 100, 3, 2},
		{100000, 1000, 2, 1},
		{100000, 1000, 3, 2},
	}

	for _, bb := range benchmarks {
		prefix := fmt.Sprintf("%d-%d", bb.accounts, bb.blockSize)
		b.Run(prefix, func(sub *testing.B) {
			benchmarkSendTxWithMultisig(sub, bb.accounts, bb.blockSize, bb.nbSigs, bb.threshold)
		})
	}
}
