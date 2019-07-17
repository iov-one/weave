package paychan

import (
	"bytes"
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/app"
	coin "github.com/iov-one/weave/coin"
	crypto "github.com/iov-one/weave/crypto"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/migration"
	"github.com/iov-one/weave/orm"
	"github.com/iov-one/weave/store"
	"github.com/iov-one/weave/weavetest"
	"github.com/iov-one/weave/x/cash"
)

var (
	now       = time.Now().UTC()
	inOneHour = now.Add(time.Hour)
)

func TestPaymentChannelHandlers(t *testing.T) {
	cashBucket := cash.NewBucket()
	bankCtrl := cash.NewController(cashBucket)
	payChanBucket := newPaymentChannelObjectBucket()
	auth := &weavetest.CtxAuth{Key: "auth"}

	rt := app.NewRouter()
	RegisterRoutes(rt, auth, bankCtrl)

	qr := weave.NewQueryRouter()
	cash.RegisterQuery(qr)
	RegisterQuery(qr)

	source := weavetest.NewCondition()
	// Because it is allowed, use different public key to sign the message.
	sourceSig := weavetest.NewKey()
	destination := weavetest.NewCondition()

	cases := map[string]struct {
		actions []action
		dbtests []querycheck
	}{
		"creating a payment channel allocates funds": {
			actions: []action{
				{
					conditions: []weave.Condition{source},
					msg: &CreateMsg{
						Metadata:     &weave.Metadata{Schema: 1},
						Source:       source.Address(),
						Destination:  destination.Address(),
						SourcePubkey: sourceSig.PublicKey(),
						Total:        dogeCoin(10, 0),
						Timeout:      weave.AsUnixTime(inOneHour),
						Memo:         "start",
					},
					blocksize: 100,
				},
			},
			dbtests: []querycheck{
				{
					path:   "/paychans",
					data:   weavetest.SequenceID(1),
					bucket: payChanBucket,
					wantRes: []orm.Object{
						orm.NewSimpleObj(weavetest.SequenceID(1), &PaymentChannel{
							Metadata:     &weave.Metadata{Schema: 1},
							Source:       source.Address(),
							Destination:  destination.Address(),
							SourcePubkey: sourceSig.PublicKey(),
							Total:        dogeCoin(10, 0),
							Timeout:      weave.AsUnixTime(inOneHour),
							Memo:         "start",
							Transferred:  dogeCoin(0, 0),
							Address:      paymentChannelAccount(weavetest.SequenceID(1)),
						}),
					},
				},
				// Query sources wallet to ensure money was
				// taken from the account.
				{
					path:   "/wallets",
					data:   source.Address(),
					bucket: cashBucket.Bucket,
					wantRes: []orm.Object{
						mustObject(cash.WalletWith(source.Address(), dogeCoin(1, 22))),
					},
				},
				// Query payment channel wallet to ensure money was
				// secured for future transactions.
				{
					path:   "/wallets",
					data:   paymentChannelAccount(weavetest.SequenceID(1)),
					bucket: cashBucket.Bucket,
					wantRes: []orm.Object{
						mustObject(cash.WalletWith(paymentChannelAccount(weavetest.SequenceID(1)), dogeCoin(10, 0))),
					},
				},
			},
		},
		"closing a channel without a transfer releases funds": {
			actions: []action{
				{
					conditions: []weave.Condition{source},
					msg: &CreateMsg{
						Metadata:     &weave.Metadata{Schema: 1},
						Source:       source.Address(),
						Destination:  destination.Address(),
						SourcePubkey: sourceSig.PublicKey(),
						Total:        dogeCoin(10, 0),
						Timeout:      weave.AsUnixTime(inOneHour),
						Memo:         "start",
					},
					blocksize: 100,
				},
				{
					conditions: []weave.Condition{}, // Timeout was reached so anyone can close it.
					msg: &CloseMsg{
						Metadata:  &weave.Metadata{Schema: 1},
						ChannelID: weavetest.SequenceID(1),
						Memo:      "end",
					},
					blocksize: 1001,
					blockTime: now.Add(123 * time.Hour),
				},
			},
			dbtests: []querycheck{
				// When payment channel is closed, it is
				// removed from the database.  Fetching an
				// entity that does not exist does not return
				// an error, but returns nil instead.
				{
					path:    "/paychans",
					data:    weavetest.SequenceID(1),
					bucket:  payChanBucket,
					wantRes: nil,
				},
				// Query sources wallet to ensure money was
				// returned to the account.
				{
					path:   "/wallets",
					data:   source.Address(),
					bucket: cashBucket.Bucket,
					wantRes: []orm.Object{
						mustObject(cash.WalletWith(source.Address(), dogeCoin(11, 22))),
					},
				},
			},
		},
		"transfer moves allocated coins to the destination": {
			actions: []action{
				{
					conditions: []weave.Condition{source},
					msg: &CreateMsg{
						Metadata:     &weave.Metadata{Schema: 1},
						Source:       source.Address(),
						Destination:  destination.Address(),
						SourcePubkey: sourceSig.PublicKey(),
						Total:        dogeCoin(10, 0),
						Timeout:      weave.AsUnixTime(inOneHour),
						Memo:         "start",
					},
					blocksize: 100,
				},
				{
					conditions: []weave.Condition{source},
					msg: setSignature(sourceSig, &TransferMsg{
						Metadata: &weave.Metadata{Schema: 1},
						Payment: &Payment{
							ChainID:   "testchain-123",
							ChannelID: weavetest.SequenceID(1),
							Amount:    dogeCoin(2, 50),
							Memo:      "much transfer",
						},
					}),
					blocksize: 103,
				},
				{
					conditions: []weave.Condition{source},
					msg: setSignature(sourceSig, &TransferMsg{
						Metadata: &weave.Metadata{Schema: 1},
						Payment: &Payment{
							ChainID:   "testchain-123",
							ChannelID: weavetest.SequenceID(1),
							Amount:    dogeCoin(3, 0),
							Memo:      "such value",
						},
					}),
					blocksize: 104,
				},
			},
			dbtests: []querycheck{
				{
					path:   "/wallets",
					data:   paymentChannelAccount(weavetest.SequenceID(1)),
					bucket: cashBucket.Bucket,
					wantRes: []orm.Object{
						mustObject(cash.WalletWith(paymentChannelAccount(weavetest.SequenceID(1)), dogeCoin(7, 0))),
					},
				},
				{
					path:   "/wallets",
					data:   destination.Address(),
					bucket: cashBucket.Bucket,
					wantRes: []orm.Object{
						mustObject(cash.WalletWith(destination.Address(), dogeCoin(3, 0))),
					},
				},
			},
		},
		"closing a channel with a transfer made releases funds": {
			actions: []action{
				{
					conditions: []weave.Condition{source},
					msg: &CreateMsg{
						Metadata:     &weave.Metadata{Schema: 1},
						Source:       source.Address(),
						Destination:  destination.Address(),
						SourcePubkey: sourceSig.PublicKey(),
						Total:        dogeCoin(10, 0),
						Timeout:      weave.AsUnixTime(inOneHour),
						Memo:         "start",
					},
					blocksize: 100,
				},
				{
					conditions: []weave.Condition{source},
					msg: setSignature(sourceSig, &TransferMsg{
						Metadata: &weave.Metadata{Schema: 1},
						Payment: &Payment{
							ChainID:   "testchain-123",
							ChannelID: weavetest.SequenceID(1),
							Amount:    dogeCoin(2, 0),
							Memo:      "much transfer",
						},
					}),
					blocksize: 104,
				},
				{
					conditions: []weave.Condition{}, // Timeout was reached so anyone can close it.
					msg: &CloseMsg{
						Metadata:  &weave.Metadata{Schema: 1},
						ChannelID: weavetest.SequenceID(1),
						Memo:      "end",
					},
					blocksize: 1001,
					blockTime: now.Add(5 * time.Hour),
				},
			},
			dbtests: []querycheck{
				// When payment channel is closed, it is
				// removed from the database. Fetching an
				// entity that does not exist does not return
				// an error, but returns nil instead.
				{
					path:    "/paychans",
					data:    weavetest.SequenceID(1),
					bucket:  payChanBucket,
					wantRes: nil,
				},
				// Query sources wallet to ensure that the
				// remaining coins were returned to the
				// account.
				{
					path:   "/wallets",
					data:   source.Address(),
					bucket: cashBucket.Bucket,
					wantRes: []orm.Object{
						mustObject(cash.WalletWith(source.Address(), dogeCoin(9, 22))),
					},
				},
				// What was transferred must belong to
				// destination.
				{
					path:   "/wallets",
					data:   destination.Address(),
					bucket: cashBucket.Bucket,
					wantRes: []orm.Object{
						mustObject(cash.WalletWith(destination.Address(), dogeCoin(2, 0))),
					},
				},
			},
		},
		"creating a payment channel without enough funds fails": {
			actions: []action{
				{
					conditions: []weave.Condition{source},
					msg: &CreateMsg{
						Metadata:     &weave.Metadata{Schema: 1},
						Source:       source.Address(),
						Destination:  destination.Address(),
						SourcePubkey: sourceSig.PublicKey(),
						Total:        dogeCoin(999, 0),
						Timeout:      weave.AsUnixTime(inOneHour),
						Memo:         "start",
					},
					blocksize:      100,
					wantDeliverErr: errors.ErrAmount,
				},
			},
		},
		"only destination can close non expired payment channel": {
			actions: []action{
				{
					conditions: []weave.Condition{source},
					msg: &CreateMsg{
						Metadata:     &weave.Metadata{Schema: 1},
						Source:       source.Address(),
						Destination:  destination.Address(),
						SourcePubkey: sourceSig.PublicKey(),
						Total:        dogeCoin(10, 0),
						Timeout:      weave.AsUnixTime(inOneHour),
						Memo:         "start",
					},
					blocksize: 100,
				},
				// Signer cannot close a channel that holds
				// funds and is not expired.
				{
					conditions: []weave.Condition{source},
					msg: &CloseMsg{
						Metadata:  &weave.Metadata{Schema: 1},
						ChannelID: weavetest.SequenceID(1),
						Memo:      "end",
					},
					blocksize:      104,
					wantDeliverErr: errors.ErrMsg,
				},
				// Destination can close channel any time.
				{
					conditions: []weave.Condition{destination},
					msg: &CloseMsg{
						Metadata:  &weave.Metadata{Schema: 1},
						ChannelID: weavetest.SequenceID(1),
						Memo:      "end",
					},
					blocksize: 108,
				},
			},
		},
		"transfer ensure transaction on the right chain": {
			actions: []action{
				{
					conditions: []weave.Condition{source},
					msg: &CreateMsg{
						Metadata:     &weave.Metadata{Schema: 1},
						Source:       source.Address(),
						Destination:  destination.Address(),
						SourcePubkey: sourceSig.PublicKey(),
						Total:        dogeCoin(10, 0),
						Timeout:      weave.AsUnixTime(inOneHour),
						Memo:         "start",
					},
					blocksize: 100,
				},
				{
					conditions: []weave.Condition{source},
					msg: setSignature(sourceSig, &TransferMsg{
						Metadata: &weave.Metadata{Schema: 1},
						Payment: &Payment{
							ChainID:   "another-chain-666",
							ChannelID: weavetest.SequenceID(1),
							Amount:    dogeCoin(2, 50),
							Memo:      "much transfer",
						},
					}),
					blocksize:    103,
					wantCheckErr: errors.ErrMsg,
				},
			},
		},
		"transfer of more funds than allocated fails": {
			actions: []action{
				{
					conditions: []weave.Condition{source},
					msg: &CreateMsg{
						Metadata:     &weave.Metadata{Schema: 1},
						Source:       source.Address(),
						Destination:  destination.Address(),
						SourcePubkey: sourceSig.PublicKey(),
						Total:        dogeCoin(10, 0),
						Timeout:      weave.AsUnixTime(inOneHour),
						Memo:         "start",
					},
					blocksize: 100,
				},
				{
					conditions: []weave.Condition{source},
					msg: setSignature(sourceSig, &TransferMsg{
						Metadata: &weave.Metadata{Schema: 1},
						Payment: &Payment{
							ChainID:   "testchain-123",
							ChannelID: weavetest.SequenceID(1),
							Amount:    dogeCoin(11, 50),
							Memo:      "much transfer",
						},
					}),
					blocksize:    103,
					wantCheckErr: errors.ErrMsg,
				},
			},
		},
		"cannot create a channel without source signature": {
			actions: []action{
				{
					conditions: []weave.Condition{source},
					msg: &CreateMsg{
						Metadata:     &weave.Metadata{Schema: 1},
						Source:       source.Address(),
						Destination:  destination.Address(),
						SourcePubkey: nil,
						Total:        dogeCoin(10, 0),
						Timeout:      weave.AsUnixTime(inOneHour),
						Memo:         "start",
					},
					blocksize:    100,
					wantCheckErr: errors.ErrMsg,
				},
			},
		},
		"transfer on a closed or non existing channel fails": {
			actions: []action{
				{
					conditions: []weave.Condition{source},
					msg: &CreateMsg{
						Metadata:     &weave.Metadata{Schema: 1},
						Source:       source.Address(),
						Destination:  destination.Address(),
						SourcePubkey: sourceSig.PublicKey(),
						Total:        dogeCoin(10, 0),
						Timeout:      weave.AsUnixTime(inOneHour),
						Memo:         "start",
					},
					blocksize: 100,
				},
				{
					conditions: []weave.Condition{destination},
					msg: &CloseMsg{
						Metadata:  &weave.Metadata{Schema: 1},
						ChannelID: weavetest.SequenceID(1),
						Memo:      "end",
					},
					blocksize: 101,
				},
				{
					conditions: []weave.Condition{source},
					msg: setSignature(sourceSig, &TransferMsg{
						Metadata: &weave.Metadata{Schema: 1},
						Payment: &Payment{
							ChainID:   "testchain-123",
							ChannelID: weavetest.SequenceID(1),
							Amount:    dogeCoin(11, 50),
							Memo:      "much transfer",
						},
					}),
					blocksize:    103,
					wantCheckErr: errors.ErrNotFound,
				},
			},
		},
		"transfer signed with invalid key fails": {
			actions: []action{
				{
					conditions: []weave.Condition{source},
					msg: &CreateMsg{
						Metadata:     &weave.Metadata{Schema: 1},
						Source:       source.Address(),
						Destination:  destination.Address(),
						SourcePubkey: sourceSig.PublicKey(),
						Total:        dogeCoin(10, 0),
						Timeout:      weave.AsUnixTime(inOneHour),
						Memo:         "start",
					},
					blocksize: 100,
				},
				{
					conditions: []weave.Condition{source},
					msg: &TransferMsg{
						Metadata: &weave.Metadata{Schema: 1},
						Signature: &crypto.Signature{
							Sig: &crypto.Signature_Ed25519{
								Ed25519: []byte("invalid signature"),
							},
						},
						Payment: &Payment{
							ChainID:   "testchain-123",
							ChannelID: weavetest.SequenceID(1),
							Amount:    dogeCoin(11, 50),
							Memo:      "much transfer",
						},
					},
					blocksize:    103,
					wantCheckErr: errors.ErrMsg,
				},
			},
		},
	}

	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			db := store.MemStore()

			migration.MustInitPkg(db, "paychan", "cash")

			// Create a source account with coins.
			wallet, err := cash.WalletWith(source.Address(), dogeCoin(11, 22))
			if err != nil {
				t.Fatalf("create wallet: %s", err)
			}
			if err := cashBucket.Save(db, wallet); err != nil {
				t.Fatalf("save wallet: %s", err)
			}

			for i, a := range tc.actions {
				cache := db.CacheWrap()
				if _, err := rt.Check(a.ctx(), cache, a.tx()); !a.wantCheckErr.Is(err) {
					t.Logf("want: %+v", a.wantCheckErr)
					t.Logf(" got: %+v", err)
					t.Fatalf("action %d check (%T)", i, a.msg)
				}
				cache.Discard()
				if a.wantCheckErr != nil {
					// Failed checks are causing the message to be ignored.
					continue
				}

				if _, err := rt.Deliver(a.ctx(), db, a.tx()); !a.wantDeliverErr.Is(err) {
					t.Logf("want: %+v", a.wantDeliverErr)
					t.Logf(" got: %+v", err)
					t.Fatalf("action %d delivery (%T)", i, a.msg)
				}
			}
			for _, tt := range tc.dbtests {
				tt.test(t, db, qr)
			}
		})
	}
}

func dogeCoin(w, f int64) *coin.Coin {
	c := coin.NewCoin(w, f, "DOGE")
	return &c
}

// action represents a single request call that is handled by a handler.
type action struct {
	conditions []weave.Condition
	msg        weave.Msg
	blocksize  int64
	// if not zero, overwrites blockTime function for timeout
	blockTime      time.Time
	wantCheckErr   *errors.Error
	wantDeliverErr *errors.Error
}

func (a *action) tx() weave.Tx {
	return &weavetest.Tx{Msg: a.msg}
}

func (a *action) ctx() weave.Context {
	ctx := weave.WithHeight(context.Background(), a.blocksize)
	ctx = weave.WithChainID(ctx, "testchain-123")
	if !a.blockTime.IsZero() {
		ctx = weave.WithBlockTime(ctx, a.blockTime)
	} else {
		ctx = weave.WithBlockTime(ctx, now)
	}
	auth := &weavetest.CtxAuth{Key: "auth"}
	return auth.SetConditions(ctx, a.conditions...)
}

// querycheck is a declaration of a query result. For given path and data
// executed within a bucket, ensure that the result is as expected.
// Make sure to register the query router.
type querycheck struct {
	path    string
	data    []byte
	bucket  orm.Bucket
	wantRes []orm.Object
}

// test ensure that querycheck declaration is the same as the database state.
func (qc *querycheck) test(t *testing.T, db weave.ReadOnlyKVStore, qr weave.QueryRouter) {
	t.Helper()

	result, err := qr.Handler(qc.path).Query(db, "", qc.data)
	if err != nil {
		t.Fatalf("query %q: %s", qc.path, err)
	}
	if w, g := len(qc.wantRes), len(result); w != g {
		t.Fatalf("want %d entries returned, got %d", w, g)
	}
	for i, wres := range qc.wantRes {
		if want, got := qc.bucket.DBKey(wres.Key()), result[i].Key; !bytes.Equal(want, got) {
			t.Errorf("want %d key to be %q, got %q", i, want, got)
		}

		if got, err := qc.bucket.Parse(nil, result[i].Value); err != nil {
			t.Errorf("parse %d: %s", i, err)
		} else if w, g := wres.Value(), got.Value(); !reflect.DeepEqual(w, g) {
			t.Logf(" got value: %+v", g)
			t.Logf("want value: %+v", w)
			t.Errorf("value %d missmatch", i)
		}
	}
}

func mustObject(obj orm.Object, err error) orm.Object {
	if err != nil {
		panic(err)
	}
	return obj
}

// setSignature computes and sets signature for given message.
func setSignature(key crypto.Signer, msg *TransferMsg) *TransferMsg {
	raw, err := msg.Payment.Marshal()
	if err != nil {
		panic(err)
	}
	sig, err := key.Sign(raw)
	if err != nil {
		panic(err)
	}
	msg.Signature = sig
	return msg
}
