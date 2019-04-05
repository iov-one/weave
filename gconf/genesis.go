package gconf

import (
	"encoding/json"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/coin"
	"github.com/iov-one/weave/errors"
)

// Initializer fulfils the InitStater interface to load data from
// the genesis file
type Initializer struct{}

var _ weave.Initializer = Initializer{}

// FromGenesis will parse initial account info from genesis
// and save it to the database
func (Initializer) FromGenesis(opts weave.Options, db weave.KVStore) error {
	conf := make(map[string]interface{})
	err := opts.ReadOptions("gconf", &conf)
	if err != nil {
		return err
	}

	for name, value := range conf {
		// All values are unmarshalled using JSON type system. This
		// means that for some values an unsupported type is selected.
		// If so, manually convert those values to the most appropriate
		// supported type.
		var err error
		switch value := value.(type) {
		case float64:
			err = SetValue(db, name, int64(value))
		case map[string]interface{}:
			// This looks like a coin!
			raw, err := json.Marshal(value)
			if err != nil {
				return errors.Wrapf(err, "cannot type cast %q value", name)
			}
			var c coin.Coin
			if err := json.Unmarshal(raw, &c); err != nil {
				return errors.Wrapf(err, "cannot type cast %q value", name)
			}
			err = SetValue(db, name, &c)
		case string:
			// AAaaa!!! string can represent an address, bytes,
			// coin or just a string
		default:
			err = SetValue(db, name, value)
		}
		if err != nil {
			return err
		}
	}

	return nil
}

// SetValue sets given value for the configuration property. Value must be one
// of supported types.
// Usually this function is not needed, because genesis allows to load all at
// once. But it comes in handy when writing tests and only few configuration
// values are necessary.
func SetValue(db weave.KVStore, propName string, value interface{}) error {
	obj, err := NewConf(propName, value)
	if err != nil {
		return errors.Wrapf(err, "cannot create %q configuration for this type of value", propName)
	}
	if err := defaultConfBucket.Save(db, obj); err != nil {
		return errors.Wrap(err, "bucket store")
	}
	return nil
}
