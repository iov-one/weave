package msgfee

import (
	"fmt"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/coin"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/gconf"
)

// Initializer fulfils the Initializer interface to load data from the genesis
// file
type Initializer struct{}

var _ weave.Initializer = (*Initializer)(nil)

// FromGenesis will parse initial account info from genesis and save it to the
// database
func (*Initializer) FromGenesis(opts weave.Options, params weave.GenesisParams, kv weave.KVStore) error {
	type msgfee struct {
		MsgPath string    `json:"msg_path"`
		Fee     coin.Coin `json:"fee"`
	}
	var fees []*msgfee
	if err := opts.ReadOptions("msgfee", &fees); err != nil {
		return errors.Wrap(err, "cannot load fees")
	}

	bucket := NewMsgFeeBucket()
	for i, f := range fees {
		fee := MsgFee{
			Metadata: &weave.Metadata{Schema: 1},
			MsgPath:  f.MsgPath,
			Fee:      f.Fee,
		}
		if err := fee.Validate(); err != nil {
			return errors.Wrap(err, fmt.Sprintf("fee #%d is invalid", i))
		}
		if _, err := bucket.Put(kv, []byte(fee.MsgPath), &fee); err != nil {
			return errors.Wrap(err, fmt.Sprintf("cannot store #%d fee", i))
		}
	}

	// We allow to initialize configuration but it is not required.
	if err := gconf.InitConfig(kv, opts, "msgfee", &Configuration{}); err != nil && !errors.ErrNotFound.Is(err) {
		return errors.Wrap(err, "init config")
	}

	return nil
}
