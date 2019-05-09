package msgfee

import (
	"testing"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/coin"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/migration"
	"github.com/iov-one/weave/store"
)

func TestMsgFeeValidate(t *testing.T) {
	cases := map[string]struct {
		mf      MsgFee
		wantErr *errors.Error
	}{
		"all good": {
			mf: MsgFee{
				Metadata: &weave.Metadata{Schema: 1},
				MsgPath:  "foo/bar",
				Fee:      coin.NewCoin(1, 2, "DOGE"),
			},
			wantErr: nil,
		},
		"missing metadata": {
			mf: MsgFee{
				MsgPath: "foo/bar",
				Fee:     coin.NewCoin(1, 2, "DOGE"),
			},
			wantErr: errors.ErrMetadata,
		},
		"empty path": {
			mf: MsgFee{
				Metadata: &weave.Metadata{Schema: 1},
				MsgPath:  "",
				Fee:      coin.NewCoin(1, 2, "DOGE"),
			},
			wantErr: errors.ErrModel,
		},
		"zero value fee with a ticker": {
			mf: MsgFee{
				Metadata: &weave.Metadata{Schema: 1},
				MsgPath:  "foo/bar",
				Fee:      coin.NewCoin(0, 0, "DOGE"),
			},
			wantErr: errors.ErrModel,
		},
		"zero value fee with no ticker": {
			mf: MsgFee{
				Metadata: &weave.Metadata{Schema: 1},
				MsgPath:  "foo/bar",
				Fee:      coin.Coin{},
			},
			wantErr: errors.ErrModel,
		},
	}

	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			err := tc.mf.Validate()
			if !tc.wantErr.Is(err) {
				t.Fatalf("got %v", err)
			}
		})
	}
}

func TestBucketMessageFee(t *testing.T) {
	b := NewMsgFeeBucket()
	db := store.MemStore()
	migration.MustInitPkg(db, "msgfee")

	_, err := b.Create(db, &MsgFee{
		Metadata: &weave.Metadata{Schema: 1},
		MsgPath:  "a/b",
		Fee:      coin.NewCoin(1, 2, "DOGE"),
	})
	if err != nil {
		t.Fatalf("cannot create a fee: %s", err)
	}

	fee, err := b.MessageFee(db, "a/b")
	if err != nil {
		t.Fatalf("got error: %v", err)
	}
	if !fee.Equals(coin.NewCoin(1, 2, "DOGE")) {
		t.Fatalf("got an unexpected fee: %v", fee)
	}

	nofee, err := b.MessageFee(db, "does-not/exist")
	if err != nil {
		t.Fatalf("got error: %v", err)
	}
	if nofee != nil {
		t.Fatalf("want nil, got %v", nofee)
	}
}
