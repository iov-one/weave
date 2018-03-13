package namecoin

import (
	"fmt"
	"testing"

	"github.com/confio/weave"
	"github.com/confio/weave/errors"
	"github.com/confio/weave/orm"
	"github.com/confio/weave/store"
	"github.com/confio/weave/x"
	"github.com/confio/weave/x/cash"
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

	addr := weave.NewAddress([]byte{1, 2, 3})
	addr2 := weave.NewAddress([]byte{4, 5, 6})

	cases := []struct {
		signers       []weave.Address
		initState     []orm.Object
		msg           weave.Msg
		expectCheck   checkErr
		expectDeliver checkErr
	}{
		0: {nil, nil, nil, errors.IsUnknownTxTypeErr, errors.IsUnknownTxTypeErr},
		1: {nil, nil, new(cash.SendMsg), cash.IsInvalidAmountErr, cash.IsInvalidAmountErr},
		2: {nil, nil, &cash.SendMsg{Amount: &foo}, errors.IsUnrecognizedAddressErr, errors.IsUnrecognizedAddressErr},
		3: {
			nil,
			nil,
			&cash.SendMsg{Amount: &foo, Src: addr, Dest: addr2},
			errors.IsUnauthorizedErr,
			errors.IsUnauthorizedErr,
		},
		// sender has no account
		4: {
			[]weave.Address{addr},
			nil,
			&cash.SendMsg{Amount: &foo, Src: addr, Dest: addr2},
			noErr, // we don't check funds
			cash.IsEmptyAccountErr,
		},
		// sender too poor
		5: {
			[]weave.Address{addr},
			[]orm.Object{mo(WalletWith(addr, "", &some))},
			&cash.SendMsg{Amount: &foo, Src: addr, Dest: addr2},
			noErr, // we don't check funds
			cash.IsInsufficientFundsErr,
		},
		// fool and his money are soon parted....
		6: {
			[]weave.Address{addr},
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

	addr := weave.NewAddress([]byte{1, 2, 3})
	addr2 := weave.NewAddress([]byte{4, 5, 6})
	addr3 := weave.NewAddress([]byte{7, 8, 9})

	ticker := "GOOD"
	msg := BuildTokenMsg(ticker, "my good token", 6)
	added := NewToken(ticker, "my good token", 6)

	// TODO: add queries to verify
	cases := []struct {
		signers       []weave.Address
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
			errors.IsUnknownTxTypeErr, errors.IsUnknownTxTypeErr, "", nil},
		// wrong currency values
		1: {nil, nil, nil, BuildTokenMsg("YO", "digga", 7),
			x.IsInvalidCurrencyErr, x.IsInvalidCurrencyErr, "", nil},
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
			errors.IsUnauthorizedErr, errors.IsUnauthorizedErr, "", nil},
		8: {[]weave.Address{addr2, addr3}, addr, nil, msg,
			errors.IsUnauthorizedErr, errors.IsUnauthorizedErr, "", nil},
		// now have permission
		9: {[]weave.Address{addr2, addr3}, addr2, nil, msg,
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

	addr := weave.NewAddress([]byte{1, 2, 3})
	addr2 := weave.NewAddress([]byte{4, 5, 6})
	// addr3 := weave.NewAddress([]byte{7, 8, 9})

	coin := x.NewCoin(100, 0, "FOO")
	name := "carl"
	// newUser + msg -> setUser
	newUser := mo(WalletWith(addr, "", &coin))
	setUser := mo(WalletWith(addr, name, &coin))
	msg := BuildSetNameMsg(addr, name)
	// dupUser already claimed this name
	dupUser := mo(WalletWith(addr2, name, &coin))

	cases := []struct {
		signer        weave.Address
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
			errors.IsUnknownTxTypeErr, errors.IsUnknownTxTypeErr, nil, nil},
		// invalid message
		1: {nil, nil, BuildSetNameMsg([]byte{1, 2}, "johnny"),
			errors.IsUnrecognizedAddressErr, errors.IsUnrecognizedAddressErr, nil, nil},
		2: {nil, nil, BuildSetNameMsg(addr, "sh"),
			IsInvalidWallet, IsInvalidWallet, nil, nil},
		// no permission to change account
		3: {nil, []orm.Object{newUser}, msg,
			errors.IsUnauthorizedErr, errors.IsUnauthorizedErr, nil, nil},
		// no account to change - only checked deliver
		4: {addr, nil, msg,
			noErr, IsInvalidWallet, nil, nil},
		5: {addr2, []orm.Object{newUser}, msg,
			errors.IsUnauthorizedErr, errors.IsUnauthorizedErr, nil, nil},
		// yes, we changed it!
		6: {addr, []orm.Object{newUser}, msg,
			noErr, noErr, addr, setUser},
		// cannot change already set - only checked deliver?
		7: {addr, []orm.Object{setUser}, msg,
			noErr, IsInvalidWallet, nil, nil},
		// cannot create conflict - only checked deliver?
		8: {addr, []orm.Object{newUser, dupUser}, msg,
			noErr, orm.IsUniqueConstraintErr, nil, nil},
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
