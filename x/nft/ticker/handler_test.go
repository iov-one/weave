package ticker_test

import (
	"crypto/rand"
	"encoding/base32"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"os"
	"path"
	"strings"
	"testing"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/app"
	"github.com/iov-one/weave/store"
	"github.com/iov-one/weave/store/iavl"
	"github.com/iov-one/weave/x"
	"github.com/iov-one/weave/x/nft"
	"github.com/iov-one/weave/x/nft/blockchain"
	"github.com/iov-one/weave/x/nft/ticker"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandleIssueTokenMsg(t *testing.T) {
	var helpers x.TestHelpers
	_, alice := helpers.MakeKey()
	_, bob := helpers.MakeKey()

	db := store.MemStore()

	nft.RegisterAction(nft.DefaultActions...)

	bucket := ticker.NewBucket()
	blockchains := blockchain.NewBucket()
	b, _ := blockchains.Create(db, alice.Address(), []byte("alicenet"), nil, blockchain.Chain{MainTickerID: []byte("IOV")}, blockchain.IOV{Codec: "asd"})
	blockchains.Save(db, b)
	o, _ := bucket.Create(db, alice.Address(), []byte("ALC0"), nil, []byte("alicenet"))
	bucket.Save(db, o)

	handler := ticker.NewIssueHandler(helpers.Authenticate(alice), nil, bucket, blockchains.Bucket)

	// when
	specs := []struct {
		owner, id       []byte
		details         ticker.TokenDetails
		approvals       []nft.ActionApprovals
		expCheckError   bool
		expDeliverError bool
	}{
		{ // happy path
			owner:   alice.Address(),
			id:      []byte("ALC1"),
			details: ticker.TokenDetails{[]byte("alicenet")},
		},
		{ // valid approvals
			owner:   alice.Address(),
			id:      []byte("ALC2"),
			details: ticker.TokenDetails{[]byte("alicenet")},
			approvals: []nft.ActionApprovals{{
				Action:    nft.UpdateDetails,
				Approvals: []nft.Approval{{Options: nft.ApprovalOptions{Count: nft.UnlimitedCount}, Address: bob.Address()}},
			}},
		},
		{ // invalid approvals
			owner:           alice.Address(),
			id:              []byte("ACL3"),
			details:         ticker.TokenDetails{[]byte("alicenet")},
			expCheckError:   true,
			expDeliverError: true,
			approvals: []nft.ActionApprovals{{
				Action:    "12",
				Approvals: []nft.Approval{{Options: nft.ApprovalOptions{}, Address: nil}},
			}},
		},
		// todo: add other test cases when details are specified
	}

	for i, spec := range specs {
		t.Run(fmt.Sprintf("case-%d", i), func(t *testing.T) {
			tx := helpers.MockTx(&ticker.IssueTokenMsg{
				Owner:     spec.owner,
				ID:        spec.id,
				Details:   spec.details,
				Approvals: spec.approvals,
			})

			// when
			cache := db.CacheWrap()
			_, err := handler.Check(nil, cache, tx)
			cache.Discard()
			if spec.expCheckError {
				require.Error(t, err)
				return
			}
			// then
			require.NoError(t, err)

			// and when delivered
			res, err := handler.Deliver(nil, db, tx)

			// then
			if spec.expDeliverError {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, res)
			assert.Equal(t, uint32(0), res.ToABCI().Code)

			// and persisted
			o, err := bucket.Get(db, spec.id)
			require.NoError(t, err)
			require.NotNil(t, o)
			u, _ := ticker.AsTicker(o)
			assert.Equal(t, spec.details.BlockchainID, u.GetBlockchainID())
			// todo: verify approvals
		})
	}
}

func TestQueryTokenByName(t *testing.T) {
	var helpers x.TestHelpers
	_, alice := helpers.MakeKey()
	_, bob := helpers.MakeKey()

	db := store.MemStore()
	bucket := ticker.NewBucket()
	o1, _ := bucket.Create(db, alice.Address(), []byte("ALC0"), nil, []byte("myBlockchainID"))
	bucket.Save(db, o1)
	o2, _ := bucket.Create(db, bob.Address(), []byte("BOB0"), nil, []byte("myOtherBlockchainID"))
	bucket.Save(db, o2)

	qr := weave.NewQueryRouter()
	ticker.RegisterQuery(qr)
	// when
	h := qr.Handler("/nft/tickers")
	require.NotNil(t, h)
	mods, err := h.Query(db, "", []byte("ALC0"))
	// then
	require.NoError(t, err)
	require.Len(t, mods, 1)

	assert.Equal(t, bucket.DBKey([]byte("ALC0")), mods[0].Key)
	got, err := bucket.Parse(nil, mods[0].Value)
	require.NoError(t, err)
	x, err := ticker.AsTicker(got)
	require.NoError(t, err)
	_ = x // todo verify stored details
}

func BenchmarkIssueToken(b *testing.B) {
	cases := []struct {
		check       bool
		deliver     bool
		txBlockSize int
	}{
		{check: true, deliver: false, txBlockSize: 10},
		{check: false, deliver: true, txBlockSize: 10},
		{check: true, deliver: true, txBlockSize: 1},
		{check: true, deliver: true, txBlockSize: 10},
		{check: true, deliver: true, txBlockSize: 100},
	}

	for _, tc := range cases {
		// Build a nice test name, considering all the parameters of a
		// table test.
		var nameChunks []string
		if tc.check {
			nameChunks = append(nameChunks, "with check")
		} else {
			nameChunks = append(nameChunks, "no check")
		}
		if tc.deliver {
			nameChunks = append(nameChunks, "with deliver")
		} else {
			nameChunks = append(nameChunks, "no deliver")
		}
		nameChunks = append(nameChunks, fmt.Sprintf("block size %d", tc.txBlockSize))
		testName := strings.Join(nameChunks, " ")

		b.Run(testName, func(b *testing.B) {
			benchIssueToken(b, tc.check, tc.deliver, tc.txBlockSize)
		})
	}
}

func benchIssueToken(
	b *testing.B,
	check bool,
	deliver bool,
	txBlockSize int,
) {
	var helpers x.TestHelpers
	_, authKey := helpers.MakeKey()

	dir := tmpDir()
	defer os.RemoveAll(dir)

	// Use commit store, so that database operations can be grouped in
	// blocks and commited in batches, just like the real application is
	// supposed to work.
	// We also use a database backend that is using a hard drive, so that
	// the benchmark is as close to a real application as possible.
	db := app.NewCommitStore(iavl.NewCommitStore(dir, b.Name()))

	tickers := ticker.NewBucket()
	blockchains := blockchain.NewBucket()
	bc, _ := blockchains.Create(db.DeliverStore(), authKey.Address(), []byte("benchnet"), nil, blockchain.Chain{MainTickerID: []byte("IOV")}, blockchain.IOV{Codec: "asd"})
	blockchains.Save(db.DeliverStore(), bc)
	handler := ticker.NewIssueHandler(helpers.Authenticate(authKey), nil, tickers, blockchains.Bucket)
	db.Commit()

	transactions := make([]weave.Tx, b.N)
	for i := range transactions {
		transactions[i] = helpers.MockTx(&ticker.IssueTokenMsg{
			Owner: authKey.Address(),
			ID:    genTickerID(i),
			Details: ticker.TokenDetails{
				BlockchainID: []byte("benchnet"),
			},
			Approvals: []nft.ActionApprovals{
				{
					Action: nft.UpdateDetails,
					Approvals: []nft.Approval{
						{Options: nft.ApprovalOptions{Count: nft.UnlimitedCount}, Address: authKey.Address()},
					},
				},
			},
		})
	}

	b.ResetTimer()

	for i, tx := range transactions {
		if check {
			_, err := handler.Check(nil, db.CheckStore(), tx)
			if err != nil {
				b.Fatalf("check %d: %s", i, err)
			}
		}

		if deliver {
			_, err := handler.Deliver(nil, db.DeliverStore(), tx)
			if err != nil {
				b.Fatalf("deliver %d: %s", i, err)
			}
		}

		// Commit only when enough transactions were processed.
		if i%txBlockSize == 0 {
			db.Commit()
		}
	}
	// Make sure buffer is cleaned up when done.
	db.Commit()
}

// genTickerID returns a unique ticker ID that is always associated with given
// number.
func genTickerID(i int) []byte {
	raw := make([]byte, 4)
	binary.LittleEndian.PutUint32(raw, uint32(i))
	id := []byte("aaaaaaaaaaaa")
	base32.StdEncoding.Encode(id, raw)
	// Ticker ID must be between 3 and 4 characters.
	return id[:4]
}

// tmpDir creates and returns a temporary directory absolute path.
func tmpDir() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		panic(b)
	}
	dir := path.Join(os.TempDir(), strings.TrimRight(base64.StdEncoding.EncodeToString(b), "="))
	if err := os.MkdirAll(dir, 0777); err != nil {
		panic("cannot created directory: " + dir)
	}
	return dir
}
