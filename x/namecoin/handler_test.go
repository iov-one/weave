package namecoin

import (
	"fmt"
	"testing"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/coin"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/migration"
	"github.com/iov-one/weave/orm"
	"github.com/iov-one/weave/store"
	"github.com/iov-one/weave/weavetest"
	"github.com/iov-one/weave/x/cash"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

	cases := []struct {
		signers       []weave.Condition
		initState     []orm.Object
		msg           weave.Msg
		expectCheck   checkErr
		expectDeliver checkErr
	}{
		0: {nil, nil, nil, errors.ErrInvalidState.Is, errors.ErrInvalidState.Is},
		1: {nil, nil, new(cash.SendMsg), errors.ErrInvalidAmount.Is, errors.ErrInvalidAmount.Is},
		2: {nil, nil, &cash.SendMsg{Amount: &foo}, errors.ErrEmpty.Is, errors.ErrEmpty.Is},
		3: {
			nil,
			nil,
			&cash.SendMsg{
				Metadata: &weave.Metadata{Schema: 1},
				Amount:   &foo,
				Src:      addr1,
				Dest:     addr2,
			},
			errors.ErrUnauthorized.Is,
			errors.ErrUnauthorized.Is,
		},
		// sender has no account
		4: {
			[]weave.Condition{perm1},
			nil,
			&cash.SendMsg{
				Metadata: &weave.Metadata{Schema: 1},
				Amount:   &foo,
				Src:      addr1,
				Dest:     addr2,
			},
			noErr, // we don't check funds
			errors.ErrEmpty.Is,
		},
		// sender too poor
		5: {
			[]weave.Condition{perm1},
			[]orm.Object{mo(WalletWith(addr1, "", &some))},
			&cash.SendMsg{
				Metadata: &weave.Metadata{Schema: 1},
				Amount:   &foo,
				Src:      addr1,
				Dest:     addr2,
			},
			noErr, // we don't check funds
			errors.ErrInsufficientAmount.Is,
		},
		// fool and his money are soon parted....
		6: {
			[]weave.Condition{perm1},
			[]orm.Object{mo(WalletWith(addr1, "fool", &foo))},
			&cash.SendMsg{
				Metadata: &weave.Metadata{Schema: 1},
				Amount:   &foo,
				Src:      addr1,
				Dest:     addr2,
			},
			noErr,
			noErr,
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("case-%d", i), func(t *testing.T) {
			auth := &weavetest.Auth{Signers: tc.signers}
			// use default controller/bucket from namecoin
			h := NewSendHandler(auth)

			kv := store.MemStore()
			migration.MustInitPkg(kv, "namecoin")
			bucket := NewWalletBucket()
			for _, wallet := range tc.initState {
				err := bucket.Save(kv, wallet)
				require.NoError(t, err)
			}

			tx := &weavetest.Tx{Msg: tc.msg}

			_, err := h.Check(nil, kv, tx)
			assert.True(t, tc.expectCheck(err), "%+v", err)
			_, err = h.Deliver(nil, kv, tx)
			assert.True(t, tc.expectDeliver(err), "%+v", err)
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
		signers       []weave.Condition
		issuer        weave.Address
		initState     []orm.Object
		msg           weave.Msg
		expectCheck   checkErr
		expectDeliver checkErr
		// query and expected are performed only if query non-empty
		query    string
		expected orm.Object
	}{
		"proper issuer - happy path": {[]weave.Condition{perm1}, addr1, nil, msg,
			noErr, noErr, ticker, added},
		"wrong message type": {[]weave.Condition{perm1}, addr1, nil, new(cash.SendMsg),
			errors.ErrInvalidAmount.Is, errors.ErrInvalidAmount.Is, "", nil},
		"invalid ticker symbol": {[]weave.Condition{perm1}, addr1, nil, BuildTokenMsg("YO", "digga", 7),
			errors.ErrCurrency.Is, errors.ErrCurrency.Is, "", nil},
		"invalid token name": {[]weave.Condition{perm1}, addr1, nil, BuildTokenMsg("GOOD", "ill3glz!", 7),
			errors.ErrInvalidInput.Is, errors.ErrInvalidInput.Is, "", nil},
		"invalid sig figs": {[]weave.Condition{perm1}, addr1, nil, BuildTokenMsg("GOOD", "my good token", 17),
			errors.ErrInvalidInput.Is, errors.ErrInvalidInput.Is, "", nil},
		"no issuer, unsigned": {nil, nil, nil, msg,
			errors.ErrUnauthorized.Is, errors.ErrUnauthorized.Is, "", nil},
		"no issuer, signed": {[]weave.Condition{perm2}, nil, nil, msg,
			errors.ErrUnauthorized.Is, errors.ErrUnauthorized.Is, "", nil},
		"cannot overwrite existing token": {[]weave.Condition{perm1}, addr1, []orm.Object{NewToken(ticker, "i was here first", 4)}, msg,
			errors.ErrDuplicate.Is, errors.ErrDuplicate.Is, "", nil},
		"can issue second token, different name": {[]weave.Condition{perm1}, addr1, []orm.Object{NewToken("OTHR", "i was here first", 4)}, msg,
			noErr, noErr, ticker, added},
		"no signature, real issuer": {nil, addr1, nil, msg,
			errors.ErrUnauthorized.Is, errors.ErrUnauthorized.Is, "", nil},
		"wrong signatures, real issuer": {[]weave.Condition{perm2, perm3}, addr1, nil, msg,
			errors.ErrUnauthorized.Is, errors.ErrUnauthorized.Is, "", nil},
		"extra signatures, real issuer": {[]weave.Condition{perm2, perm3}, addr2, nil, msg,
			noErr, noErr, ticker, added},
	}

	for testname, tc := range cases {
		t.Run(testname, func(t *testing.T) {
			auth := &weavetest.Auth{Signers: tc.signers}
			// use default controller/bucket from namecoin
			h := NewTokenHandler(auth, tc.issuer)

			db := store.MemStore()
			migration.MustInitPkg(db, "namecoin")
			bucket := NewTokenBucket()
			for _, wallet := range tc.initState {
				err := bucket.Save(db, wallet)
				require.NoError(t, err)
			}

			tx := &weavetest.Tx{Msg: tc.msg}

			// note that this counts on checkDB *not* creating it
			_, err := h.Check(nil, db, tx)
			assert.True(t, tc.expectCheck(err), "%+v", err)
			_, err = h.Deliver(nil, db, tx)
			assert.True(t, tc.expectDeliver(err), "%+v", err)

			if tc.query != "" {
				res, err := bucket.Get(db, tc.query)
				require.NoError(t, err)
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

	cases := []struct {
		signer        weave.Condition
		initState     []orm.Object
		msg           weave.Msg
		expectCheck   checkErr
		expectDeliver checkErr
		// query and expected are performed only after successful deliver
		query    weave.Address
		expected orm.Object
	}{
		// wrong message type
		0: {nil, nil, new(cash.SendMsg),
			errors.ErrInvalidAmount.Is, errors.ErrInvalidAmount.Is, nil, nil},
		// invalid message
		1: {nil, nil, BuildSetNameMsg([]byte{1, 2}, "johnny"),
			errors.ErrInvalidInput.Is, errors.ErrInvalidInput.Is, nil, nil},
		2: {nil, nil, BuildSetNameMsg(addr1, "sh"),
			errors.ErrInvalidInput.Is, errors.ErrInvalidInput.Is, nil, nil},
		// no permission to change account
		3: {nil, []orm.Object{newUser}, msg,
			errors.ErrUnauthorized.Is, errors.ErrUnauthorized.Is, nil, nil},
		// no account to change - only checked deliver
		4: {perm1, nil, msg,
			noErr, errors.ErrNotFound.Is, nil, nil},
		5: {perm2, []orm.Object{newUser}, msg,
			errors.ErrUnauthorized.Is, errors.ErrUnauthorized.Is, nil, nil},
		// yes, we changed it!
		6: {perm1, []orm.Object{newUser}, msg,
			noErr, noErr, addr1, setUser},
		// cannot change already set - only checked deliver?
		7: {perm1, []orm.Object{setUser}, msg,
			noErr, errors.ErrCannotBeModified.Is, nil, nil},
		// cannot create conflict - only checked deliver?
		8: {perm1, []orm.Object{newUser, dupUser}, msg,
			noErr, errors.ErrDuplicate.Is, nil, nil},
		// cannot change - no such a wallet (should should up by addr2 not addr1)
		9: {perm1, []orm.Object{dupUser}, msg, noErr,
			func(err error) bool { return errors.ErrNotFound.Is(err) },
			addr1, nil},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("case-%d", i), func(t *testing.T) {
			auth := &weavetest.Auth{Signer: tc.signer}
			// use default controller/bucket from namecoin
			bucket := NewWalletBucket()
			h := NewSetNameHandler(auth, bucket)

			db := store.MemStore()
			migration.MustInitPkg(db, "namecoin")
			for _, wallet := range tc.initState {
				err := bucket.Save(db, wallet)
				require.NoError(t, err)
			}

			tx := &weavetest.Tx{Msg: tc.msg}

			// note that this counts on checkDB *not* creating it
			_, err := h.Check(nil, db, tx)
			assert.True(t, tc.expectCheck(err), "%+v", err)
			_, err = h.Deliver(nil, db, tx)
			assert.True(t, tc.expectDeliver(err), "%+v", err)

			if tc.query != nil {
				res, err := bucket.Get(db, tc.query)
				require.NoError(t, err)
				assert.Equal(t, tc.expected, res)
			}
		})
	}
}
