package cash

import (
	"github.com/confio/weave"
	"github.com/confio/weave/x"
)

var (
	cfgPrefix = []byte("_c:")
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

// Get loads the config from the db or the cache
func (c *Configure) Get(db weave.KVStore) *Config {
	// only load once
	if c.cache == nil {
		value := db.Get(c.key())
		if value == nil {
			err := c.Set(db, c.base)
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

// Set will store the configuration, returning an error if it
// is invalid
func (c *Configure) Set(db weave.KVStore, config *Config) error {
	bz, err := x.MarshalValid(config)
	if err != nil {
		return err
	}
	db.Set(c.key(), bz)
	return nil
}

func (c *Configure) key() []byte {
	return append(cfgPrefix, []byte(c.name)...)
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
		return c.MinFee.Validate()
	}
	return nil
}

// DefaultConfig returns a default config where the fees go to
// a burn address and no min fee is enforced
func DefaultConfig() *Config {
	return &Config{
		Collector: weave.NewAddress([]byte("no-fees-here")),
	}
}
