package namecoin

import (
	"fmt"
	"testing"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/orm"
	"github.com/iov-one/weave/store"
	"github.com/iov-one/weave/x"
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
	var helpers x.TestHelpers

	foo := x.NewCoin(100, 0, "FOO")
	some := x.NewCoin(300, 0, "SOME")

	perm := weave.NewCondition("sig", "ed25519", []byte{1, 2, 3})
	perm2 := weave.NewCondition("sig", "ed25519", []byte{4, 5, 6})
	addr := perm.Address()
	addr2 := perm2.Address()

	cases := []struct {
		signers       []weave.Condition
		initState     []orm.Object
		msg           weave.Msg
		expectCheck   checkErr
		expectDeliver checkErr
	}{
		0: {nil, nil, nil, errors.ErrInvalidMsg.Is, errors.ErrInvalidMsg.Is},
		1: {nil, nil, new(cash.SendMsg), errors.ErrInvalidAmount.Is, errors.ErrInvalidAmount.Is},
		2: {nil, nil, &cash.SendMsg{Amount: &foo}, errors.IsUnrecognizedAddressErr, errors.IsUnrecognizedAddressErr},
		3: {
			nil,
			nil,
			&cash.SendMsg{Amount: &foo, Src: addr, Dest: addr2},
			errors.ErrUnauthorized.Is,
			errors.ErrUnauthorized.Is,
		},
		// sender has no account
		4: {
			[]weave.Condition{perm},
			nil,
			&cash.SendMsg{Amount: &foo, Src: addr, Dest: addr2},
			noErr, // we don't check funds
			errors.ErrEmpty.Is,
		},
		// sender too poor
		5: {
			[]weave.Condition{perm},
			[]orm.Object{mo(WalletWith(addr, "", &some))},
			&cash.SendMsg{Amount: &foo, Src: addr, Dest: addr2},
			noErr, // we don't check funds
			errors.ErrInsufficientAmount.Is,
		},
		// fool and his money are soon parted....
		6: {
			[]weave.Condition{perm},
			[]orm.Object{mo(WalletWith(addr, "fool", &foo))},
			&cash.SendMsg{Amount: &foo, Src: addr, Dest: addr2},
			noErr,
			noErr,
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("case-%d", i), func(t *testing.T) {
			auth := helpers.Authenticate(tc.signers...)
			// use default controller/bucket from namecoin
			h := NewSendHandler(auth)

			kv := store.MemStore()
			bucket := NewWalletBucket()
			for _, wallet := range tc.initState {
				err := bucket.Save(kv, wallet)
				require.NoError(t, err)
			}

			tx := helpers.MockTx(tc.msg)

			_, err := h.Check(nil, kv, tx)
			assert.True(t, tc.expectCheck(err), "%+v", err)
			_, err = h.Deliver(nil, kv, tx)
			assert.True(t, tc.expectDeliver(err), "%+v", err)
		})
	}
}

func TestNewTokenHandler(t *testing.T) {
	var helpers x.TestHelpers

	_, perm := helpers.MakeKey()
	_, perm2 := helpers.MakeKey()
	_, perm3 := helpers.MakeKey()
	addr := perm.Address()
	addr2 := perm2.Address()

	ticker := "GOOD"
	msg := BuildTokenMsg(ticker, "my good token", 6)
	added := NewToken(ticker, "my good token", 6)

	// TODO: add queries to verify
	cases := []struct {
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
		// wrong message type
		0: {nil, nil, nil, new(cash.SendMsg),
			errors.ErrInvalidMsg.Is, errors.ErrInvalidMsg.Is, "", nil},
		// wrong currency values
		1: {nil, nil, nil, BuildTokenMsg("YO", "digga", 7),
			x.ErrInvalidCurrency.Is, x.ErrInvalidCurrency.Is, "", nil},
		2: {nil, nil, nil, BuildTokenMsg("GOOD", "ill3glz!", 7),
			IsInvalidToken, IsInvalidToken, "", nil},
		3: {nil, nil, nil, BuildTokenMsg("GOOD", "my good token", 17),
			IsInvalidToken, IsInvalidToken, "", nil},
		// valid message, done!
		4: {nil, nil, nil, msg,
			noErr, noErr, ticker, added},
		// try to overwrite
		5: {nil, nil, []orm.Object{NewToken(ticker, "i was here first", 4)}, msg,
			IsInvalidToken, IsInvalidToken, "", nil},
		// different name is fine
		6: {nil, nil, []orm.Object{NewToken("OTHR", "i was here first", 4)}, msg,
			noErr, noErr, ticker, added},
		// not enough permissions
		7: {nil, addr, nil, msg,
			errors.ErrUnauthorized.Is, errors.ErrUnauthorized.Is, "", nil},
		8: {[]weave.Condition{perm2, perm3}, addr, nil, msg,
			errors.ErrUnauthorized.Is, errors.ErrUnauthorized.Is, "", nil},
		// now have permission
		9: {[]weave.Condition{perm2, perm3}, addr2, nil, msg,
			noErr, noErr, ticker, added},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("case-%d", i), func(t *testing.T) {
			auth := helpers.Authenticate(tc.signers...)
			// use default controller/bucket from namecoin
			h := NewTokenHandler(auth, tc.issuer)

			db := store.MemStore()
			bucket := NewTokenBucket()
			for _, wallet := range tc.initState {
				err := bucket.Save(db, wallet)
				require.NoError(t, err)
			}

			tx := helpers.MockTx(tc.msg)

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
	var helpers x.TestHelpers

	_, perm := helpers.MakeKey()
	_, perm2 := helpers.MakeKey()
	addr := perm.Address()
	addr2 := perm2.Address()

	coin := x.NewCoin(100, 0, "FOO")
	name := "carl"
	// newUser + msg -> setUser
	newUser := mo(WalletWith(addr, "", &coin))
	setUser := mo(WalletWith(addr, name, &coin))
	msg := BuildSetNameMsg(addr, name)
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
			errors.ErrInvalidMsg.Is, errors.ErrInvalidMsg.Is, nil, nil},
		// invalid message
		1: {nil, nil, BuildSetNameMsg([]byte{1, 2}, "johnny"),
			errors.IsUnrecognizedAddressErr, errors.IsUnrecognizedAddressErr, nil, nil},
		2: {nil, nil, BuildSetNameMsg(addr, "sh"),
			IsInvalidWallet, IsInvalidWallet, nil, nil},
		// no permission to change account
		3: {nil, []orm.Object{newUser}, msg,
			errors.ErrUnauthorized.Is, errors.ErrUnauthorized.Is, nil, nil},
		// no account to change - only checked deliver
		4: {perm, nil, msg,
			noErr, IsInvalidWallet, nil, nil},
		5: {perm2, []orm.Object{newUser}, msg,
			errors.ErrUnauthorized.Is, errors.ErrUnauthorized.Is, nil, nil},
		// yes, we changed it!
		6: {perm, []orm.Object{newUser}, msg,
			noErr, noErr, addr, setUser},
		// cannot change already set - only checked deliver?
		7: {perm, []orm.Object{setUser}, msg,
			noErr, IsInvalidWallet, nil, nil},
		// cannot create conflict - only checked deliver?
		8: {perm, []orm.Object{newUser, dupUser}, msg,
			noErr, errors.ErrDuplicate.Is, nil, nil},
		// cannot change - no such a wallet (should should up by addr2 not addr)
		9: {perm, []orm.Object{dupUser}, msg, noErr,
			func(err error) bool { return errors.IsSameError(err, ErrNoSuchWallet(addr)) },
			addr, nil},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("case-%d", i), func(t *testing.T) {
			auth := helpers.Authenticate()
			if tc.signer != nil {
				auth = helpers.Authenticate(tc.signer)
			}

			// use default controller/bucket from namecoin
			bucket := NewWalletBucket()
			h := NewSetNameHandler(auth, bucket)

			db := store.MemStore()
			for _, wallet := range tc.initState {
				err := bucket.Save(db, wallet)
				require.NoError(t, err)
			}

			tx := helpers.MockTx(tc.msg)

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
