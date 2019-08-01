package server

import (
	"encoding/json"
	"io/ioutil"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/store"
)

// ValidateGenesis TODO
func ValidateGenesis(ini weave.Initializer, genesisPaths []string) error {
	for _, path := range genesisPaths {
		if err := validateGenesis(ini, path); err != nil {
			return errors.Wrap(err, path)
		}
	}
	return nil
}

func validateGenesis(ini weave.Initializer, genesisPath string) error {
	b, err := ioutil.ReadFile(genesisPath)
	if err != nil {
		return errors.Wrap(err, "cannot read genesis file")
	}

	var genesis struct {
		State weave.Options `json:"app_state"`
	}
	if err := json.Unmarshal(b, &genesis); err != nil {
		return errors.Wrap(err, "cannot JSON deserialize genesis")
	}

	// Use in memory store because we want to discard the result.
	db := store.MemStore()

	if err := ini.FromGenesis(genesis.State, weave.GenesisParams{}, db); err != nil {
		return errors.Wrap(err, "cannot initialize from genesis")
	}

	return nil
}
