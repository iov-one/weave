package paychan

import (
	"bytes"
	"context"
	"encoding/binary"
	"reflect"
	"testing"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/app"
	crypto "github.com/iov-one/weave/crypto"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/orm"
	"github.com/iov-one/weave/store"
	"github.com/iov-one/weave/x"
	"github.com/iov-one/weave/x/cash"
)

var helper x.TestHelpers

func TestPaymentChannelHandlers(t *testing.T) {
	cashBucket := cash.NewBucket()
	bankCtrl := cash.NewController(cashBucket)
	payChanBucket := NewPaymentChannelBucket()
	auth := helper.CtxAuth("auth")

	rt := app.NewRouter()
	RegisterRoutes(rt, auth, bankCtrl)

	qr := weave.NewQueryRouter()
	cash.RegisterQuery(qr)
	RegisterQuery(qr)

	_, src := helper.MakeKey()
	// Because it is allowed, use different public key to sign the message.
	srcSig, _ := helper.MakeKey()
	_, recipient := helper.MakeKey()

	cases := map[string]struct {
		actions []action
		dbtests []querycheck
	}{
		"creating a payment channel allocates funds": {
			actions: []action{
				{
					conditions: []weave.Condition{src},
					msg: &CreatePaymentChannelMsg{
						Src:          src.Address(),
						Recipient:    recipient.Address(),
						SenderPubkey: srcSig.PublicKey(),
						Total:        dogeCoin(10, 0),
						Timeout:      1000,
						Memo:         "start",
					},
					blocksize: 100,
				},
			},
			dbtests: []querycheck{
				{
					path:   "/paychans",
					data:   asSeqID(1),
					bucket: payChanBucket.Bucket,
					wantRes: []orm.Object{
						orm.NewSimpleObj(asSeqID(1), &PaymentChannel{
							Src:          src.Address(),
							Recipient:    recipient.Address(),
							SenderPubkey: srcSig.PublicKey(),
							Total:        dogeCoin(10, 0),
							Timeout:      1000,
							Memo:         "start",
							Transferred:  dogeCoin(0, 0),
						}),
					},
				},
				// Query senders wallet to ensure money was
				// taken from the account.
				{
					path:   "/wallets",
					data:   src.Address(),
					bucket: cashBucket.Bucket,
					wantRes: []orm.Object{
						mustObject(cash.WalletWith(src.Address(), dogeCoin(1, 22))),
					},
				},
				// Query payment channel wallet to ensure money was
				// secured for future transactions.
				{
					path:   "/wallets",
					data:   paymentChannelAccount(asSeqID(1)),
					bucket: cashBucket.Bucket,
					wantRes: []orm.Object{
						mustObject(cash.WalletWith(paymentChannelAccount(asSeqID(1)), dogeCoin(10, 0))),
					},
				},
			},
		},
		"closing a channel without a transfer releases funds": {
			actions: []action{
				{
					conditions: []weave.Condition{src},
					msg: &CreatePaymentChannelMsg{
						Src:          src.Address(),
						Recipient:    recipient.Address(),
						SenderPubkey: srcSig.PublicKey(),
						Total:        dogeCoin(10, 0),
						Timeout:      1000,
						Memo:         "start",
					},
					blocksize: 100,
				},
				{
					conditions: []weave.Condition{src},
					msg: &ClosePaymentChannelMsg{
						ChannelID: asSeqID(1),
						Memo:      "end",
					},
					blocksize: 1001,
				},
			},
			dbtests: []querycheck{
				// When payment channel is closed, it is
				// removed from the database.  Fetching an
				// entity that does not exist does not return
				// an error, but returns nil instead.
				{
					path:    "/paychans",
					data:    asSeqID(1),
					bucket:  payChanBucket.Bucket,
					wantRes: nil,
				},
				// Query senders wallet to ensure money was
				// returned to the account.
				{
					path:   "/wallets",
					data:   src.Address(),
					bucket: cashBucket.Bucket,
					wantRes: []orm.Object{
						mustObject(cash.WalletWith(src.Address(), dogeCoin(11, 22))),
					},
				},
			},
		},
		"transfer moves allocated coins to the recipient": {
			actions: []action{
				{
					conditions: []weave.Condition{src},
					msg: &CreatePaymentChannelMsg{
						Src:          src.Address(),
						Recipient:    recipient.Address(),
						SenderPubkey: srcSig.PublicKey(),
						Total:        dogeCoin(10, 0),
						Timeout:      1000,
						Memo:         "start",
					},
					blocksize: 100,
				},
				{
					conditions: []weave.Condition{src},
					msg: setSignature(srcSig, &TransferPaymentChannelMsg{
						Payment: &Payment{
							ChainID:   "testchain-123",
							ChannelID: asSeqID(1),
							Amount:    dogeCoin(2, 50),
							Memo:      "much transfer",
						},
					}),
					blocksize: 103,
				},
				{
					conditions: []weave.Condition{src},
					msg: setSignature(srcSig, &TransferPaymentChannelMsg{
						Payment: &Payment{
							ChainID:   "testchain-123",
							ChannelID: asSeqID(1),
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
					data:   paymentChannelAccount(asSeqID(1)),
					bucket: cashBucket.Bucket,
					wantRes: []orm.Object{
						mustObject(cash.WalletWith(paymentChannelAccount(asSeqID(1)), dogeCoin(7, 0))),
					},
				},
				{
					path:   "/wallets",
					data:   recipient.Address(),
					bucket: cashBucket.Bucket,
					wantRes: []orm.Object{
						mustObject(cash.WalletWith(recipient.Address(), dogeCoin(3, 0))),
					},
				},
			},
		},
		"closing a channel with a transfer made releases funds": {
			actions: []action{
				{
					conditions: []weave.Condition{src},
					msg: &CreatePaymentChannelMsg{
						Src:          src.Address(),
						Recipient:    recipient.Address(),
						SenderPubkey: srcSig.PublicKey(),
						Total:        dogeCoin(10, 0),
						Timeout:      1000,
						Memo:         "start",
					},
					blocksize: 100,
				},
				{
					conditions: []weave.Condition{src},
					msg: setSignature(srcSig, &TransferPaymentChannelMsg{
						Payment: &Payment{
							ChainID:   "testchain-123",
							ChannelID: asSeqID(1),
							Amount:    dogeCoin(2, 0),
							Memo:      "much transfer",
						},
					}),
					blocksize: 104,
				},
				{
					conditions: []weave.Condition{src},
					msg: &ClosePaymentChannelMsg{
						ChannelID: asSeqID(1),
						Memo:      "end",
					},
					blocksize: 1001,
				},
			},
			dbtests: []querycheck{
				// When payment channel is closed, it is
				// removed from the database. Fetching an
				// entity that does not exist does not return
				// an error, but returns nil instead.
				{
					path:    "/paychans",
					data:    asSeqID(1),
					bucket:  payChanBucket.Bucket,
					wantRes: nil,
				},
				// Query senders wallet to ensure that the
				// remaining coins were returned to the
				// account.
				{
					path:   "/wallets",
					data:   src.Address(),
					bucket: cashBucket.Bucket,
					wantRes: []orm.Object{
						mustObject(cash.WalletWith(src.Address(), dogeCoin(9, 22))),
					},
				},
				// What was transferred must belong to
				// recipient.
				{
					path:   "/wallets",
					data:   recipient.Address(),
					bucket: cashBucket.Bucket,
					wantRes: []orm.Object{
						mustObject(cash.WalletWith(recipient.Address(), dogeCoin(2, 0))),
					},
				},
			},
		},
		"creating a payment channel without enough funds fails": {
			actions: []action{
				{
					conditions: []weave.Condition{src},
					msg: &CreatePaymentChannelMsg{
						Src:          src.Address(),
						Recipient:    recipient.Address(),
						SenderPubkey: srcSig.PublicKey(),
						Total:        dogeCoin(999, 0),
						Timeout:      1000,
						Memo:         "start",
					},
					blocksize:      100,
					wantDeliverErr: errors.ErrInsufficientAmount.New("funds"),
				},
			},
		},
		"only recipient can close non expired payment channel": {
			actions: []action{
				{
					conditions: []weave.Condition{src},
					msg: &CreatePaymentChannelMsg{
						Src:          src.Address(),
						Recipient:    recipient.Address(),
						SenderPubkey: srcSig.PublicKey(),
						Total:        dogeCoin(10, 0),
						Timeout:      500,
						Memo:         "start",
					},
					blocksize: 100,
				},
				// Signer cannot close a channel that holds
				// funds and is not expired.
				{
					conditions: []weave.Condition{src},
					msg: &ClosePaymentChannelMsg{
						ChannelID: asSeqID(1),
						Memo:      "end",
					},
					blocksize:      104,
					wantDeliverErr: errors.ErrInvalidMsg,
				},
				// Recipient can close channel any time.
				{
					conditions: []weave.Condition{recipient},
					msg: &ClosePaymentChannelMsg{
						ChannelID: asSeqID(1),
						Memo:      "end",
					},
					blocksize: 108,
				},
			},
		},
		"transfer ensure transaction on the right chain": {
			actions: []action{
				{
					conditions: []weave.Condition{src},
					msg: &CreatePaymentChannelMsg{
						Src:          src.Address(),
						Recipient:    recipient.Address(),
						SenderPubkey: srcSig.PublicKey(),
						Total:        dogeCoin(10, 0),
						Timeout:      1000,
						Memo:         "start",
					},
					blocksize: 100,
				},
				{
					conditions: []weave.Condition{src},
					msg: setSignature(srcSig, &TransferPaymentChannelMsg{
						Payment: &Payment{
							ChainID:   "another-chain-666",
							ChannelID: asSeqID(1),
							Amount:    dogeCoin(2, 50),
							Memo:      "much transfer",
						},
					}),
					blocksize:    103,
					wantCheckErr: errors.ErrInvalidMsg,
				},
			},
		},
		"transfer of more funds than allocated fails": {
			actions: []action{
				{
					conditions: []weave.Condition{src},
					msg: &CreatePaymentChannelMsg{
						Src:          src.Address(),
						Recipient:    recipient.Address(),
						SenderPubkey: srcSig.PublicKey(),
						Total:        dogeCoin(10, 0),
						Timeout:      1000,
						Memo:         "start",
					},
					blocksize: 100,
				},
				{
					conditions: []weave.Condition{src},
					msg: setSignature(srcSig, &TransferPaymentChannelMsg{
						Payment: &Payment{
							ChainID:   "testchain-123",
							ChannelID: asSeqID(1),
							Amount:    dogeCoin(11, 50),
							Memo:      "much transfer",
						},
					}),
					blocksize:    103,
					wantCheckErr: errors.ErrInvalidMsg,
				},
			},
		},
		"cannot create a channel without sender signature": {
			actions: []action{
				{
					conditions: []weave.Condition{src},
					msg: &CreatePaymentChannelMsg{
						Src:          src.Address(),
						Recipient:    recipient.Address(),
						SenderPubkey: nil,
						Total:        dogeCoin(10, 0),
						Timeout:      1000,
						Memo:         "start",
					},
					blocksize:    100,
					wantCheckErr: errors.ErrInvalidMsg,
				},
			},
		},
		"transfer on a closed or non existing channel fails": {
			actions: []action{
				{
					conditions: []weave.Condition{src},
					msg: &CreatePaymentChannelMsg{
						Src:          src.Address(),
						Recipient:    recipient.Address(),
						SenderPubkey: srcSig.PublicKey(),
						Total:        dogeCoin(10, 0),
						Timeout:      1000,
						Memo:         "start",
					},
					blocksize: 100,
				},
				{
					conditions: []weave.Condition{recipient},
					msg: &ClosePaymentChannelMsg{
						ChannelID: asSeqID(1),
						Memo:      "end",
					},
					blocksize: 101,
				},
				{
					conditions: []weave.Condition{src},
					msg: setSignature(srcSig, &TransferPaymentChannelMsg{
						Payment: &Payment{
							ChainID:   "testchain-123",
							ChannelID: asSeqID(1),
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
					conditions: []weave.Condition{src},
					msg: &CreatePaymentChannelMsg{
						Src:          src.Address(),
						Recipient:    recipient.Address(),
						SenderPubkey: srcSig.PublicKey(),
						Total:        dogeCoin(10, 0),
						Timeout:      1000,
						Memo:         "start",
					},
					blocksize: 100,
				},
				{
					conditions: []weave.Condition{src},
					msg: &TransferPaymentChannelMsg{
						Signature: &crypto.Signature{
							Sig: &crypto.Signature_Ed25519{
								Ed25519: []byte("invalid signature"),
							},
						},
						Payment: &Payment{
							ChainID:   "testchain-123",
							ChannelID: asSeqID(1),
							Amount:    dogeCoin(11, 50),
							Memo:      "much transfer",
						},
					},
					blocksize:    103,
					wantCheckErr: errors.ErrInvalidMsg,
				},
			},
		},
	}

	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			db := store.MemStore()

			// Create a sender account with coins.
			wallet, err := cash.WalletWith(src.Address(), dogeCoin(11, 22))
			if err != nil {
				t.Fatalf("create wallet: %s", err)
			}
			if err := cashBucket.Save(db, wallet); err != nil {
				t.Fatalf("save wallet: %s", err)
			}

			for i, a := range tc.actions {
				cache := db.CacheWrap()
				if _, err := rt.Check(a.ctx(), cache, a.tx()); !errors.Is(err, a.wantCheckErr) {
					t.Logf("want: %+v", a.wantCheckErr)
					t.Logf(" got: %+v", err)
					t.Fatalf("action %d check (%T)", i, a.msg)
				}
				cache.Discard()
				if a.wantCheckErr != nil {
					// Failed checks are causing the message to be ignored.
					continue
				}

				if _, err := rt.Deliver(a.ctx(), db, a.tx()); !errors.Is(err, a.wantDeliverErr) {
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

// asSeqID returns an ID encoded as if it was generated by the bucket sequence
// call.
func asSeqID(i int64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(i))
	return b
}

func dogeCoin(w, f int64) *coin.Coin {
	c := x.NewCoin(w, f, "DOGE")
	return &c
}

// action represents a single request call that is handled by a handler.
type action struct {
	conditions     []weave.Condition
	msg            weave.Msg
	blocksize      int64
	wantCheckErr   error
	wantDeliverErr error
}

func (a *action) tx() weave.Tx {
	return helper.MockTx(a.msg)
}

func (a *action) ctx() weave.Context {
	ctx := weave.WithHeight(context.Background(), a.blocksize)
	ctx = weave.WithChainID(ctx, "testchain-123")
	return helper.CtxAuth("auth").SetConditions(ctx, a.conditions...)
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
func setSignature(key crypto.Signer, msg *TransferPaymentChannelMsg) *TransferPaymentChannelMsg {
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
