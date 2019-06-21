package namecoin

import (
	"testing"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/coin"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/migration"
	"github.com/iov-one/weave/orm"
	"github.com/iov-one/weave/store"
	"github.com/iov-one/weave/weavetest"
	"github.com/iov-one/weave/weavetest/assert"
	"github.com/iov-one/weave/x/cash"
)

type checkErr func(error) bool

func noErr(err error) bool { return err == nil }

// mo = must Object
func mo(obj orm.Object, err error) orm.Object {
	if err != nil {
		panic(err)
	}
	return obj
}

// TestSendHandler lightly adapted from x/cash to make sure code still
// works with our new bucket implementation
func TestSendHandler(t *testing.T) {
	foo := coin.NewCoin(100, 0, "FOO")
	some := coin.NewCoin(300, 0, "SOME")

	perm1 := weave.NewCondition("sig", "ed25519", []byte{1, 2, 3})
	perm2 := weave.NewCondition("sig", "ed25519", []byte{4, 5, 6})
	addr1 := perm1.Address()
	addr2 := perm2.Address()

	cases := map[string]struct {
		signers     []weave.Condition
		initState   []orm.Object
		msg         weave.Msg
		wantCheck   *errors.Error
		wantDeliver *errors.Error
	}{
		"success": {
			signers: []weave.Condition{perm1},
			initState: []orm.Object{
				mo(WalletWith(addr1, "fool", &foo)),
			},
			msg: &cash.SendMsg{
				Metadata: &weave.Metadata{Schema: 1},
				Amount:   &foo,
				Src:      addr1,
				Dest:     addr2,
			},
			wantCheck:   nil,
			wantDeliver: nil,
		},
		"nil message": {
			wantCheck:   errors.ErrState,
			wantDeliver: errors.ErrState,
		},
		"empty message": {
			msg:         &cash.SendMsg{},
			wantCheck:   errors.ErrAmount,
			wantDeliver: errors.ErrAmount,
		},
		"invalid message": {
			msg:         &cash.SendMsg{Amount: &foo},
			wantCheck:   errors.ErrEmpty,
			wantDeliver: errors.ErrEmpty,
		},
		"missing signature": {
			msg: &cash.SendMsg{
				Metadata: &weave.Metadata{Schema: 1},
				Amount:   &foo,
				Src:      addr1,
				Dest:     addr2,
			},
			wantCheck:   errors.ErrUnauthorized,
			wantDeliver: errors.ErrUnauthorized,
		},
		"sender has no account": {
			signers: []weave.Condition{perm1},
			msg: &cash.SendMsg{
				Metadata: &weave.Metadata{Schema: 1},
				Amount:   &foo,
				Src:      addr1,
				Dest:     addr2,
			},
			wantCheck:   nil,
			wantDeliver: errors.ErrEmpty,
		},
		"sender too poor": {
			signers: []weave.Condition{perm1},
			initState: []orm.Object{
				mo(WalletWith(addr1, "", &some)),
			},
			msg: &cash.SendMsg{
				Metadata: &weave.Metadata{Schema: 1},
				Amount:   &foo,
				Src:      addr1,
				Dest:     addr2,
			},
			wantCheck:   nil,
			wantDeliver: errors.ErrAmount,
		},
	}

	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			auth := &weavetest.Auth{Signers: tc.signers}
			// Use default controller/bucket from namecoin.
			h := NewSendHandler(auth)

			kv := store.MemStore(179)
			migration.MustInitPkg(kv, "namecoin")
			bucket := NewWalletBucket()
			for i, wallet := range tc.initState {
				if err := bucket.Save(kv, wallet); err != nil {
					t.Fatalf("cannot initialize state: wallet %d: %s", i, err)
				}
			}

			tx := &weavetest.Tx{Msg: tc.msg}
			if _, err := h.Check(nil, kv, tx); !tc.wantCheck.Is(err) {
				t.Fatalf("unexpected check result: %s", err)
			}
			if _, err := h.Deliver(nil, kv, tx); !tc.wantDeliver.Is(err) {
				t.Fatalf("unexpected deliver result: %s", err)
			}
		})
	}
}

func TestNewTokenHandler(t *testing.T) {
	perm1 := weavetest.NewCondition()
	perm2 := weavetest.NewCondition()
	perm3 := weavetest.NewCondition()
	addr1 := perm1.Address()
	addr2 := perm2.Address()

	ticker := "GOOD"
	msg := BuildTokenMsg(ticker, "my good token", 6)
	added := NewToken(ticker, "my good token", 6)

	// TODO: add queries to verify
	cases := map[string]struct {
		signers     []weave.Condition
		issuer      weave.Address
		initState   []orm.Object
		msg         weave.Msg
		wantCheck   *errors.Error
		wantDeliver *errors.Error
		// query and expected are performed only if query non-empty
		query    string
		expected orm.Object
	}{
		"proper issuer - happy path": {
			signers:  []weave.Condition{perm1},
			issuer:   addr1,
			msg:      msg,
			query:    ticker,
			expected: added,
		},
		"wrong message type": {
			signers:     []weave.Condition{perm1},
			issuer:      addr1,
			msg:         &cash.SendMsg{},
			wantCheck:   errors.ErrAmount,
			wantDeliver: errors.ErrAmount,
		},
		"invalid ticker symbol": {
			signers:     []weave.Condition{perm1},
			issuer:      addr1,
			msg:         BuildTokenMsg("YO", "digga", 7),
			wantCheck:   errors.ErrCurrency,
			wantDeliver: errors.ErrCurrency,
		},
		"invalid token name": {
			signers:     []weave.Condition{perm1},
			issuer:      addr1,
			msg:         BuildTokenMsg("GOOD", "ill3glz!", 7),
			wantCheck:   errors.ErrInput,
			wantDeliver: errors.ErrInput,
		},
		"invalid sig figs": {
			signers:     []weave.Condition{perm1},
			issuer:      addr1,
			msg:         BuildTokenMsg("GOOD", "my good token", 17),
			wantCheck:   errors.ErrInput,
			wantDeliver: errors.ErrInput,
		},
		"no issuer, unsigned": {
			msg:         msg,
			wantCheck:   errors.ErrUnauthorized,
			wantDeliver: errors.ErrUnauthorized,
		},
		"no issuer, signed": {
			signers:     []weave.Condition{perm2},
			msg:         msg,
			wantCheck:   errors.ErrUnauthorized,
			wantDeliver: errors.ErrUnauthorized,
		},
		"cannot overwrite existing token": {
			signers: []weave.Condition{perm1},
			issuer:  addr1,
			initState: []orm.Object{
				NewToken(ticker, "i was here first", 4),
			},
			msg:         msg,
			wantCheck:   errors.ErrDuplicate,
			wantDeliver: errors.ErrDuplicate,
		},
		"can issue second token, different name": {
			signers: []weave.Condition{perm1},
			issuer:  addr1,
			initState: []orm.Object{
				NewToken("OTHR", "i was here first", 4),
			},
			msg:      msg,
			query:    ticker,
			expected: added,
		},
		"no signature, real issuer": {
			issuer:      addr1,
			msg:         msg,
			wantCheck:   errors.ErrUnauthorized,
			wantDeliver: errors.ErrUnauthorized,
		},
		"wrong signatures, real issuer": {
			signers:     []weave.Condition{perm2, perm3},
			issuer:      addr1,
			msg:         msg,
			wantCheck:   errors.ErrUnauthorized,
			wantDeliver: errors.ErrUnauthorized,
		},
		"extra signatures, real issuer": {
			signers:  []weave.Condition{perm2, perm3},
			issuer:   addr2,
			msg:      msg,
			query:    ticker,
			expected: added,
		},
	}

	for testname, tc := range cases {
		t.Run(testname, func(t *testing.T) {
			auth := &weavetest.Auth{Signers: tc.signers}
			// Use default controller/bucket from namecoin.
			h := NewTokenHandler(auth, tc.issuer)

			db := store.MemStore(179)
			migration.MustInitPkg(db, "namecoin")
			bucket := NewTokenBucket()
			for i, wallet := range tc.initState {
				if err := bucket.Save(db, wallet); err != nil {
					t.Fatalf("cannot initialize state: wallet %d: %s", i, err)
				}
			}

			tx := &weavetest.Tx{Msg: tc.msg}
			if _, err := h.Check(nil, db, tx); !tc.wantCheck.Is(err) {
				t.Fatalf("unexpected check result: %s", err)
			}
			if _, err := h.Deliver(nil, db, tx); !tc.wantDeliver.Is(err) {
				t.Fatalf("unexpected deliver result: %s", err)
			}

			if tc.query != "" {
				res, err := bucket.Get(db, tc.query)
				if err != nil {
					t.Fatalf("bucket lookup failed: %s", err)
				}
				assert.Equal(t, tc.expected, res)
			}
		})
	}
}

func TestSetNameHandler(t *testing.T) {
	perm1 := weavetest.NewCondition()
	perm2 := weavetest.NewCondition()
	addr1 := perm1.Address()
	addr2 := perm2.Address()

	coin := coin.NewCoin(100, 0, "FOO")
	name := "carl"
	// newUser + msg -> setUser
	newUser := mo(WalletWith(addr1, "", &coin))
	setUser := mo(WalletWith(addr1, name, &coin))
	msg := BuildSetNameMsg(addr1, name)
	// dupUser already claimed this name
	dupUser := mo(WalletWith(addr2, name, &coin))

	cases := map[string]struct {
		signer      weave.Condition
		initState   []orm.Object
		msg         weave.Msg
		wantCheck   *errors.Error
		wantDeliver *errors.Error
		// query and expected are performed only after successful deliver
		query    weave.Address
		expected orm.Object
	}{
		"success": {
			signer:    perm1,
			initState: []orm.Object{newUser},
			msg:       msg,
			query:     addr1,
			expected:  setUser,
		},
		"wrong message type": {
			msg:         &cash.SendMsg{},
			wantCheck:   errors.ErrAmount,
			wantDeliver: errors.ErrAmount,
		},
		"invalid message": {
			msg:         BuildSetNameMsg([]byte{1, 2}, "johnny"),
			wantCheck:   errors.ErrInput,
			wantDeliver: errors.ErrInput,
		},
		"invalid message data": {
			msg:         BuildSetNameMsg(addr1, "sh"),
			wantCheck:   errors.ErrInput,
			wantDeliver: errors.ErrInput,
		},
		"no permission to change account": {
			initState:   []orm.Object{newUser},
			msg:         msg,
			wantCheck:   errors.ErrUnauthorized,
			wantDeliver: errors.ErrUnauthorized,
		},
		"no account to change - only checked deliver": {
			signer:      perm1,
			msg:         msg,
			wantDeliver: errors.ErrNotFound,
		},
		"not authorized": {
			signer:      perm2,
			initState:   []orm.Object{newUser},
			msg:         msg,
			wantCheck:   errors.ErrUnauthorized,
			wantDeliver: errors.ErrUnauthorized,
		},
		"cannot change already set - only checked deliver?": {
			signer:      perm1,
			initState:   []orm.Object{setUser},
			msg:         msg,
			wantDeliver: errors.ErrImmutable,
		},
		"cannot create conflict - only checked deliver?": {
			signer:      perm1,
			initState:   []orm.Object{newUser, dupUser},
			msg:         msg,
			wantDeliver: errors.ErrDuplicate,
		},
		"cannot change - no such a wallet (should should up by addr2 not addr1)": {
			signer:      perm1,
			initState:   []orm.Object{dupUser},
			msg:         msg,
			wantDeliver: errors.ErrNotFound,
			query:       addr1,
		},
	}

	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			auth := &weavetest.Auth{Signer: tc.signer}
			// Use default controller/bucket from namecoin.
			bucket := NewWalletBucket()
			h := NewSetNameHandler(auth, bucket)

			db := store.MemStore(179)
			migration.MustInitPkg(db, "namecoin")
			for i, wallet := range tc.initState {
				if err := bucket.Save(db, wallet); err != nil {
					t.Fatalf("cannot initialize state: wallet %d: %s", i, err)
				}
			}

			tx := &weavetest.Tx{Msg: tc.msg}
			if _, err := h.Check(nil, db, tx); !tc.wantCheck.Is(err) {
				t.Fatalf("unexpected check result: %s", err)
			}
			if _, err := h.Deliver(nil, db, tx); !tc.wantDeliver.Is(err) {
				t.Fatalf("unexpected deliver result: %s", err)
			}

			if tc.query != nil {
				res, err := bucket.Get(db, tc.query)
				if err != nil {
					t.Fatalf("bucket lookup failed: %s", err)
				}
				assert.Equal(t, tc.expected, res)
			}
		})
	}
}
