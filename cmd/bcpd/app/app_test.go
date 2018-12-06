package app

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"testing"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/app"
	"github.com/iov-one/weave/crypto"
	"github.com/iov-one/weave/x"
	"github.com/iov-one/weave/x/batch"
	"github.com/iov-one/weave/x/cash"
	"github.com/iov-one/weave/x/escrow"
	"github.com/iov-one/weave/x/multisig"
	"github.com/iov-one/weave/x/sigs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	abci "github.com/tendermint/tendermint/abci/types"
	cmn "github.com/tendermint/tendermint/libs/common"
	"github.com/tendermint/tendermint/libs/log"
)

func TestSendTx(t *testing.T) {
	chainID := "test-net-22"
	mainAccount := &account{pk: crypto.GenPrivKeyEd25519()}
	myApp := newTestApp(t, chainID, []*account{mainAccount})

	// Query for my balance
	key := cash.NewBucket().DBKey(mainAccount.address())
	queryAndCheckWallet(t, myApp, "/", key, cash.Set{
		Coins: x.Coins{
			{Ticker: "ETH", Whole: 50000},
			{Ticker: "FRNK", Whole: 1234},
		},
	})

	// build and sign a transaction
	pk2 := crypto.GenPrivKeyEd25519()
	addr2 := pk2.PublicKey().Address()
	dres := sendBatch(t, false, myApp, chainID, 2, []*account{mainAccount}, mainAccount.address(), addr2, 2000, "ETH", "Have a great trip!")

	// ensure 3 keys with proper values
	if assert.Equal(t, 3, len(dres.Tags), "%#v", dres.Tags) {
		addr := mainAccount.pk.PublicKey().Address()
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
	queryAndCheckWallet(t, myApp, "/", key, cash.Set{
		Coins: x.Coins{
			{Ticker: "ETH", Whole: 30000},
			{Ticker: "FRNK", Whole: 1234},
		},
	})

	dres = sendBatch(t, true, myApp, chainID, 2, []*account{mainAccount}, mainAccount.address(), addr2, 2000, "ETH", "Have a great trip!")
	assert.NotEqual(t, 0, dres.Code)
}

func toHex(s string) string {
	h := hex.EncodeToString([]byte(s))
	return strings.ToUpper(h)
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
	key := cash.NewBucket().DBKey(mainAccount.address())
	queryAndCheckWallet(t, myApp, "/", key, cash.Set{
		Coins: x.Coins{
			{Ticker: "ETH", Whole: 48000},
			{Ticker: "FRNK", Whole: 1234},
		},
	})

	// make sure money arrived safely
	key2 := cash.NewBucket().DBKey(addr2)
	queryAndCheckWallet(t, myApp, "/", key2, cash.Set{
		Coins: x.Coins{
			{Ticker: "ETH", Whole: 2000},
		},
	})

	// make sure other paths also get this value....
	queryAndCheckWallet(t, myApp, "/wallets", addr2, cash.Set{
		Coins: x.Coins{
			{Ticker: "ETH", Whole: 2000},
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
	appState := withWalletAppState(t, accounts)

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

func newMultisigTestApp(t require.TestingT, chainID string, contracts []*contract) app.BaseApp {
	// no minimum fee, in-memory data-store
	abciApp, err := GenerateApp("", log.NewNopLogger(), true)
	require.NoError(t, err)
	myApp := abciApp.(app.BaseApp) // let's set up a genesis file with some cash
	appState := withContractAppState(t, contracts)

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

func withWalletAppState(t require.TestingT, accounts []*account) string {
	type wallet struct {
		Address weave.Address `json:"address"`
		Coins   x.Coins       `json:"coins"`
	}
	var state struct {
		Cash []wallet `json:"cash"`
	}

	for _, acc := range accounts {
		state.Cash = append(state.Cash, wallet{
			Address: acc.address(),
			Coins: x.Coins{
				&x.Coin{Whole: 50000, Ticker: "ETH"},
				&x.Coin{Whole: 1234, Ticker: "FRNK"},
			},
		})
	}
	raw, err := json.MarshalIndent(state, "", "  ")
	assert.NoErrorf(t, err, "marshal state: %s", err)
	return string(raw)
}

type contract struct {
	id          []byte
	accountSigs []*account
	threshold   int64
}

func (c *contract) address() []byte {
	return multisig.MultiSigCondition(c.id).Address()
}

func (c *contract) sigs() [][]byte {
	var sigsAddr = make([][]byte, len(c.accountSigs))

	for i, s := range c.accountSigs {
		sigsAddr[i] = s.address()
	}

	return sigsAddr
}

func (c *contract) signers() []*account {
	return c.accountSigs[:c.threshold]
}

func withContractAppState(t require.TestingT, contracts []*contract) string {
	var buff bytes.Buffer
	for i, acc := range contracts {
		_, err := buff.WriteString(fmt.Sprintf(`{
            "name": "wallet%d",
			"address": "%X",
			"coins": [{
                "whole": 50000,
                "ticker": "ETH"
            },{
                "whole": 1234,
				"ticker": "FRNK"
			}]
		},`, i, acc.address()))
		require.NoError(t, err)
	}

	walletStr := buff.String()
	walletStr = walletStr[:len(walletStr)-1]
	appState := fmt.Sprintf(`{
		"wallets": [%s],
        "currencies": [{
            "ticker": "ETH",
            "name": "Smells like ethereum",
            "sig_figs": 9
        },{
            "ticker": "FRNK",
            "name": "Frankie",
            "sig_figs": 3
		}]
	}`, walletStr)

	return appState
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

	res := signAndCommit(t, false, baseApp, tx, signers, chainID, height)

	// make sure money arrived safely
	queryAndCheckWallet(t, baseApp, "/wallets", to, cash.Set{
		Coins: x.Coins{
			{Ticker: ticker, Whole: amount},
		},
	})

	return res
}

// checks batchWorks
func sendBatch(t require.TestingT, fail bool, baseApp app.BaseApp, chainID string, height int64, signers []*account,
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

	var messages []BatchMsg_Union
	for i := 0; i < batch.MaxBatchMessages; i++ {
		messages = append(messages,
			BatchMsg_Union{
				Sum: &BatchMsg_Union_SendMsg{
					SendMsg: msg,
				},
			})
	}

	if fail == true {
		messages[0] = BatchMsg_Union{
			Sum: &BatchMsg_Union_CreateEscrowMsg{
				&escrow.CreateEscrowMsg{},
			},
		}
	}

	tx := &Tx{
		Sum:      createBatchMsg(messages),
		Multisig: contracts,
	}

	res := signAndCommit(t, fail, baseApp, tx, signers, chainID, height)

	// make sure money arrived safely
	if !fail {
		queryAndCheckWallet(t, baseApp, "/wallets", to, cash.Set{
			Coins: x.Coins{
				{Ticker: ticker, Whole: amount * batch.MaxBatchMessages},
			},
		})
	}

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

	dres := signAndCommit(t, false, baseApp, tx, signers, chainID, height)

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
func signAndCommit(t require.TestingT, fail bool, app app.BaseApp, tx *Tx, signers []*account, chainID string,
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
	if !fail {
		require.Equal(t, uint32(0), chres.Code, chres.Log)
	}
	dres := app.DeliverTx(txBytes)

	if !fail {
		require.Equal(t, uint32(0), dres.Code, dres.Log)
	}

	app.EndBlock(abci.RequestEndBlock{})
	cres := app.Commit()
	assert.NotEmpty(t, cres.Data)
	return dres
}

// queryAndCheckWallet queries the wallet from the chain and check it is the one expected
func queryAndCheckWallet(t require.TestingT, baseApp app.BaseApp, path string, data []byte, expected cash.Set) {
	query := abci.RequestQuery{Path: path, Data: data}
	res := baseApp.Query(query)

	// check query was ok
	require.Equal(t, uint32(0), res.Code, "%#v", res)
	assert.NotEmpty(t, res.Value)

	var actual cash.Set
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

// makeSendTx is a special case of sendToken when the sender account is the only signer
// this is used in our benchmark
func makeSendTx(t require.TestingT, chainID string, sender, receiver *account, ticker, memo string, amount int64) []byte {
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
		Sum: &Tx_SendMsg{msg},
	}

	sig, err := sigs.SignTx(sender.pk, tx, chainID, sender.nonce())
	require.NoError(t, err)
	tx.Signatures = append(tx.Signatures, sig)

	txBytes, err := tx.Marshal()
	require.NoError(t, err)
	require.NotEmpty(t, txBytes)
	return txBytes
}

// makeSendTxMultisig is a special case of sendToken when the sender and receiver accounts are a contract
// this is used in our benchmark
func makeSendTxMultisig(t require.TestingT, chainID string, sender, receiver *contract, ticker, memo string, amount int64) []byte {
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
		Multisig: [][]byte{sender.id},
	}

	mandatorySigners := sender.signers()
	for _, acc := range mandatorySigners {
		sig, err := sigs.SignTx(acc.pk, tx, chainID, acc.nonce())
		require.NoError(t, err)
		tx.Signatures = append(tx.Signatures, sig)
	}

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

// benchmarkSendTxWithMultisig runs the actual benchmark sequence with multisig eg.
// N * CheckTx
// BeginBlock
// N * DeliverTx
// EndBlock
// Commit
func benchmarkSendTxWithMultisig(b *testing.B, nbAccounts, blockSize, nbContracts, nbMultisigSigs int, threshold int64) {
	id := func(i int64) []byte {
		bz := make([]byte, 8)
		binary.BigEndian.PutUint64(bz, uint64(i))
		return bz
	}

	accounts := make([]*account, nbAccounts)
	for i := 0; i < nbAccounts; i++ {
		accounts[i] = &account{pk: crypto.GenPrivKeyEd25519()}
	}

	contracts := make([]*contract, nbContracts)
	for i := 0; i < nbContracts; i++ {
		sigs := make([]*account, nbMultisigSigs)
		for k := 0; k < nbMultisigSigs; k++ {
			sigs[k] = accounts[cmn.RandInt()%nbAccounts]
		}
		contracts[i] = &contract{id: id(int64(i + 1)), accountSigs: sigs, threshold: threshold}
	}

	chainID := "bench-net-22"
	myApp := newMultisigTestApp(b, chainID, contracts)

	for i, c := range contracts {
		signer := accounts[cmn.RandInt()%nbAccounts]
		c.id = createContract(b, myApp, chainID, int64(i+1), []*account{signer}, threshold, c.sigs()...)
	}

	txs := make([][]byte, b.N)
	for i := 0; i < b.N; i++ {
		sender := contracts[cmn.RandInt()%nbContracts]
		recipient := contracts[cmn.RandInt()%nbContracts]
		txs[i] = makeSendTxMultisig(b, chainID, sender, recipient, "ETH", "benchmark", 1)
	}

	b.ResetTimer()

	// iterate through Txs
	// start at one to not trigger block creation at the first iteration
	height := 0
	for i := 1; i <= b.N; i++ {
		chres := myApp.CheckTx(txs[i-1])
		require.Equal(b, uint32(0), chres.Code, chres.Log)

		// if there is no remainder we have enough txs to create a block
		// If the number of tx is not a multiple of the block size, we create
		// a final block at the end with the remaining txs
		if i%blockSize == 0 || i == b.N {
			height++
			header := abci.Header{Height: int64(height + nbContracts + 1)}
			myApp.BeginBlock(abci.RequestBeginBlock{Header: header})

			// deliver from the first tx following previous block (or 0) to the current tx
			for k := (height - 1) * blockSize; k < i; k++ {
				dres := myApp.DeliverTx(txs[k])
				require.Equal(b, uint32(0), dres.Code, dres.Log)
			}

			myApp.EndBlock(abci.RequestEndBlock{})
			cres := myApp.Commit()
			assert.NotEmpty(b, cres.Data)
		}
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

	// iterate through Txs
	// start at one to not trigger block creation at the first iteration
	height := 0
	for i := 1; i <= b.N; i++ {
		chres := myApp.CheckTx(txs[i-1])
		require.Equal(b, uint32(0), chres.Code, chres.Log)

		// if there is no remainder we have enough txs to create a block
		// If the number of tx is not a multiple of the block size, we create
		// a final block at the end with the remaining txs
		if i%blockSize == 0 || i == b.N {
			height++
			header := abci.Header{Height: int64(height + 1)}
			myApp.BeginBlock(abci.RequestBeginBlock{Header: header})

			for k := (height - 1) * blockSize; k < i; k++ {
				dres := myApp.DeliverTx(txs[k])
				require.Equal(b, uint32(0), dres.Code, dres.Log)
			}

			// deliver from the first tx following previous block (or 0) to the current tx
			myApp.EndBlock(abci.RequestEndBlock{})
			cres := myApp.Commit()
			assert.NotEmpty(b, cres.Data)
		}
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
		contracts int
		nbSigs    int
		threshold int64
	}{
		{10000, 100, 100, 2, 1},
		{10000, 100, 100, 10, 5},
		{10000, 100, 1000, 2, 1},
		{10000, 100, 1000, 10, 5},
		{10000, 100, 1000, 20, 10},
		{10000, 1000, 100, 2, 1},
		{10000, 1000, 100, 10, 5},
		{10000, 1000, 100, 20, 10},
	}

	for _, bb := range benchmarks {
		prefix := fmt.Sprintf("%d-%d-%d-%d-%d", bb.accounts, bb.blockSize, bb.contracts, bb.nbSigs, bb.threshold)
		b.Run(prefix, func(sub *testing.B) {
			benchmarkSendTxWithMultisig(sub, bb.accounts, bb.blockSize, bb.contracts, bb.nbSigs, bb.threshold)
		})
	}
}

func createBatchMsg(messages []BatchMsg_Union) *Tx_BatchMsg {
	return &Tx_BatchMsg{
		BatchMsg: &BatchMsg{
			Messages: messages,
		},
	}
}
