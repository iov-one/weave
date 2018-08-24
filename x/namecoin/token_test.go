package namecoin

import (
	"fmt"
	"testing"

	"github.com/confio/weave"
	"github.com/confio/weave/orm"
	"github.com/confio/weave/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type saver interface {
	Save(weave.KVStore, orm.Object) error
}

func saveAll(b saver, db weave.KVStore, objs []orm.Object) error {
	for _, obj := range objs {
		err := b.Save(db, obj)
		if err != nil {
			return err
		}
	}
	return nil
}

func TestTokenBucket(t *testing.T) {
	bucket := NewTokenBucket()
	addr := weave.NewAddress([]byte{1, 2, 3, 4})

	cases := []struct {
		set      []orm.Object
		setError bool
		queries  []string
		expected []*Token
	}{
		// empty
		0: {nil, false, []string{"HELP"}, []*Token{nil}},
		// reject wrong type
		1: {[]orm.Object{NewWallet(addr)}, true, nil, nil},
		// reject invalid tokens
		2: {[]orm.Object{NewToken("LONGER", "My name", 4)}, true, nil, nil},
		3: {[]orm.Object{NewToken("SHRT", "My name", 18)}, true, nil, nil},
		4: {[]orm.Object{NewToken("SHRT", "Cr*z! as F*c!", 4)}, true, nil, nil},
		// query works fine with one or two tokens
		5: {
			[]orm.Object{NewToken("ABC", "Michael", 5)},
			false,
			[]string{"ABC", "LED"},
			[]*Token{&Token{"Michael", 5}, nil},
		},
		6: {
			[]orm.Object{
				NewToken("ABC", "Jackson", 5),
				NewToken("LED", "Zeppelin", 4),
			},
			false,
			[]string{"ABC", "LED"},
			[]*Token{&Token{"Jackson", 5}, &Token{"Zeppelin", 4}},
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("case-%d", i), func(t *testing.T) {
			db := store.MemStore()
			err := saveAll(bucket, db, tc.set)
			if tc.setError {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			for j, q := range tc.queries {
				token, err := bucket.Get(db, q)
				require.NoError(t, err)
				if token != nil {
					assert.EqualValues(t, q, AsTicker(token))
				}
				assert.EqualValues(t, tc.expected[j], AsToken(token), q)
			}
		})
	}
}
