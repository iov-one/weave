package cash

import (
	"github.com/confio/weave"
	"github.com/confio/weave/x"
)

// Configure stores the configuration in the kv-store and
type Configure struct {
	name  string
	base  *Config
	cache *Config
}

// NewConfigure sets up a Config holder, storing data based
// on name and using base if nothing is stored in db
func NewConfigure(name string, base *Config) *Configure {
	return &Configure{
		name:  name,
		base:  base,
		cache: nil,
	}
}

// MinFee will get a Coin struct (not pointer) from the config
func (c *Configure) MinFee(db weave.KVStore) x.Coin {
	cfg := c.get(db).(*Config)
	if cfg.MinFee == nil {
		return x.Coin{}
	}
	return *cfg.MinFee
}

// Collector will get the collector as an address from the Config
func (c *Configure) Collector(db weave.KVStore) weave.Address {
	cfg := c.get(db).(*Config)
	return cfg.Collector
}

// Set will store the configuration, returning an error if it
// is invalid
func (c *Configure) Set(db weave.KVStore, config *Config) error {
	return c.set(db, config)
}

//--------------------- Config ------

// Validate ensures that
func (c *Config) Validate() error {
	err := weave.Address(c.Collector).Validate()
	if err != nil {
		return err
	}
	// validate a MinFee if it exists
	if c.MinFee != nil {
		return c.MinFee.ValidOrEmpty()
	}
	return nil
}

// DefaultConfig returns a default config where the fees go to
// a burn address and no min fee is enforced
func DefaultConfig(min *x.Coin) *Config {
	return &Config{
		MinFee:    min,
		Collector: weave.NewAddress([]byte("no-fees-here")),
	}
}

//-------- internals can be refactored out....

var (
	cfgPrefix = []byte("_c:")
)

func (c *Configure) set(db weave.KVStore, config x.MarshalValidater) error {
	bz, err := x.MarshalValid(config)
	if err != nil {
		return err
	}
	db.Set(c.key(), bz)
	return nil
}

// get loads the config from the db or the cache
func (c *Configure) get(db weave.KVStore) weave.Persistent {
	// only load once
	if c.cache == nil {
		value := db.Get(c.key())
		if value == nil {
			err := c.set(db, c.base)
			if err != nil {
				// default must always be valid!
				panic(err)
			}
			c.cache = c.base
		} else {
			c.cache = new(Config)
			x.MustUnmarshal(c.cache, value)
		}
	}
	return c.cache
}

func (c *Configure) key() []byte {
	return append(cfgPrefix, []byte(c.name)...)
}
