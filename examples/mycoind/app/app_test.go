package app

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"testing"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/app"
	"github.com/iov-one/weave/coin"
	"github.com/iov-one/weave/commands/server"
	"github.com/iov-one/weave/crypto"
	"github.com/iov-one/weave/migration"
	"github.com/iov-one/weave/weavetest/assert"
	"github.com/iov-one/weave/x/cash"
	"github.com/iov-one/weave/x/sigs"
	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/libs/log"
)

func testInitChain(t *testing.T, myApp app.BaseApp, addr string) {
	t.Helper()

	chainID := "test-net-22"

	// Local Alias for JSON types, so that declaration is nicer.
	type (
		dict  map[string]interface{}
		array []interface{}
	)
	appState, err := json.Marshal(dict{
		"cash": array{
			dict{
				"address": addr,
				"coins": array{
					dict{"whole": 50000, "ticker": "ETH"},
					dict{"whole": 1234, "ticker": "FRNK"},
				},
			},
		},
		"conf": dict{
			"cash": cash.Configuration{
				CollectorAddress: fromHex(t, "3b11c732b8fc1f09beb34031302fe2ab347c5c14"),
				MinimalFee:       coin.Coin{Whole: 0}, // no fee
			},
			"migration": migration.Configuration{
				Admin: fromHex(t, "3b11c732b8fc1f09beb34031302fe2ab347c5c14"),
			},
		},
		"initialize_schema": []dict{
			dict{"ver": 1, "pkg": "batch"},
			dict{"ver": 1, "pkg": "cash"},
			dict{"ver": 1, "pkg": "currency"},
			dict{"ver": 1, "pkg": "distribution"},
			dict{"ver": 1, "pkg": "escrow"},
			dict{"ver": 1, "pkg": "gov"},
			dict{"ver": 1, "pkg": "msgfee"},
			dict{"ver": 1, "pkg": "multisig"},
			dict{"ver": 1, "pkg": "namecoin"},
			dict{"ver": 1, "pkg": "nft"},
			dict{"ver": 1, "pkg": "paychan"},
			dict{"ver": 1, "pkg": "sigs"},
			dict{"ver": 1, "pkg": "utils"},
			dict{"ver": 1, "pkg": "validators"},
		},
	})
	if err != nil {
		t.Fatalf("cannot serialize state: %s", err)
	}
	assert.Equal(t, "", myApp.GetChainID())
	myApp.InitChain(abci.RequestInitChain{
		AppStateBytes: appState,
		ChainId:       chainID,
	})
	assert.Equal(t, chainID, myApp.GetChainID())

}

func fromHex(t testing.TB, s string) weave.Address {
	t.Helper()
	raw, err := hex.DecodeString(s)
	if err != nil {
		t.Fatalf("cannot decode hex encoded address %q: %s", s, err)
	}
	return raw
}

// testCommit will commit at height h and return new hash
func testCommit(t *testing.T, myApp app.BaseApp, h int64) []byte {
	// Commit first block, make sure non-nil hash
	header := abci.Header{Height: h}
	myApp.BeginBlock(abci.RequestBeginBlock{Header: header})
	myApp.EndBlock(abci.RequestEndBlock{})
	cres := myApp.Commit()
	hash := cres.Data
	if len(hash) == 0 {
		t.Fatalf("hash value must not be empty")
	}
	return hash
}

func testQuery(t *testing.T, myApp app.BaseApp, path string, key []byte, obj weave.Persistent) {
	// Query for my balance
	query := abci.RequestQuery{
		Path: path,
		Data: key,
	}
	qres := myApp.Query(query)
	assert.Equal(t, uint32(0), qres.Code)
	if len(qres.Value) == 0 {
		t.Fatalf("query value must not be empty")
	}
	if path == "/" && len(key)+2 != len(qres.Key) {
		// the original key will be embedded in a result set
		// this should add two bytes to it
		t.Fatalf("key too long %x", qres.Key)
	}
	// unpack the ResultSet
	// parse it and check it is not empty
	err := app.UnmarshalOneResult(qres.Value, obj)
	assert.Nil(t, err)
}

func testSendTx(t *testing.T, myApp app.BaseApp, h int64,
	amount int64, ticker string,
	sender *crypto.PrivateKey, rcpt weave.Address, seq int64) abci.ResponseDeliverTx {

	msg := &cash.SendMsg{
		Metadata: &weave.Metadata{Schema: 1},
		Src:      sender.PublicKey().Address(),
		Dest:     rcpt,
		Amount: &coin.Coin{
			Whole:  amount,
			Ticker: ticker,
		},
		Memo: "Have a great trip!",
	}
	tx := &Tx{
		Sum: &Tx_SendMsg{msg},
	}
	sig, err := sigs.SignTx(sender, tx, myApp.GetChainID(), seq)
	assert.Nil(t, err)
	tx.Signatures = []*sigs.StdSignature{sig}
	txBytes, err := tx.Marshal()
	assert.Nil(t, err)
	if len(txBytes) == 0 {
		t.Fatalf("txBytes must not be empty")
	}

	// Submit to the chain
	header := abci.Header{Height: h}
	myApp.BeginBlock(abci.RequestBeginBlock{Header: header})
	// check and deliver must pass
	chres := myApp.CheckTx(txBytes)
	assert.Equal(t, uint32(0), chres.Code)
	dres := myApp.DeliverTx(txBytes)
	assert.Equal(t, uint32(0), dres.Code)
	return dres
}

func TestApp(t *testing.T) {
	// no minimum fee, in-memory data-store
	opts := &server.Options{
		MinFee: coin.Coin{},
		Home:   "",
		Logger: log.NewNopLogger(),
		Debug:  true,
	}
	abciApp, err := GenerateApp(opts)
	assert.Nil(t, err)
	myApp := abciApp.(app.BaseApp)

	// let's set up a genesis file with some cash
	pk := crypto.GenPrivKeyEd25519()
	addr := pk.PublicKey().Address()

	testInitChain(t, myApp, addr.String())
	hash1 := testCommit(t, myApp, 1)

	var acct cash.Set
	key := cash.NewBucket().DBKey(addr)
	testQuery(t, myApp, "/", key, &acct)
	assert.Equal(t, 2, len(acct.Coins))
	assert.Equal(t, int64(50000), acct.Coins[0].Whole)
	assert.Equal(t, "FRNK", acct.Coins[1].Ticker)

	// build and sign a transaction
	pk2 := crypto.GenPrivKeyEd25519()
	addr2 := pk2.PublicKey().Address()
	dres := testSendTx(t, myApp, 2, 2000, "ETH", pk, addr2, 0)
	// and commit the block
	hash2 := testCommit(t, myApp, 2)
	if bytes.Equal(hash1, hash2) {
		t.Fatal("Hash must change after commit")
	}

	// ensure 3 keys with proper values
	assert.Equal(t, 3, len(dres.Tags))
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

	// Query for new balances (same key, new state)
	var acct2 cash.Set
	testQuery(t, myApp, "/", key, &acct2)
	assert.Equal(t, 2, len(acct2.Coins))
	assert.Equal(t, int64(48000), acct2.Coins[0].Whole)
	assert.Equal(t, int64(1234), acct2.Coins[1].Whole)

	// make sure money arrived safely
	var acct3 cash.Set
	key2 := cash.NewBucket().DBKey(addr2)
	testQuery(t, myApp, "/", key2, &acct3)
	assert.Equal(t, 1, len(acct3.Coins))
	assert.Equal(t, int64(2000), acct3.Coins[0].Whole)
	assert.Equal(t, "ETH", acct3.Coins[0].Ticker)

	// make sure other paths also get this value....
	var acct4 cash.Set
	testQuery(t, myApp, "/wallets", addr2, &acct4)
	assert.Equal(t, 1, len(acct4.Coins))
	assert.Equal(t, int64(2000), acct4.Coins[0].Whole)
	assert.Equal(t, "ETH", acct4.Coins[0].Ticker)

	// prefix scan works
	var acct5 cash.Set
	testQuery(t, myApp, "/wallets?prefix", addr, &acct5)
	assert.Equal(t, 2, len(acct2.Coins))
	assert.Equal(t, int64(48000), acct2.Coins[0].Whole)
	assert.Equal(t, int64(1234), acct2.Coins[1].Whole)

	// try another send
	testSendTx(t, myApp, 3, 100, "FRNK", pk, addr2, 1)
	// and commit the block
	hash3 := testCommit(t, myApp, 3)
	if bytes.Equal(hash2, hash3) {
		t.Fatal("Hash must change after commit")
	}

	var second cash.Set
	testQuery(t, myApp, "/wallets", addr2, &second)
	assert.Equal(t, 2, len(second.Coins))
	assert.Equal(t, int64(2000), second.Coins[0].Whole)
	assert.Equal(t, "ETH", second.Coins[0].Ticker)
	assert.Equal(t, int64(100), second.Coins[1].Whole)
	assert.Equal(t, "FRNK", second.Coins[1].Ticker)
}
