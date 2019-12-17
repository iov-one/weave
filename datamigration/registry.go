package datamigration

import (
	"context"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
)

// MustRegister registers an initialization function for given entity.
func MustRegister(migrationID string, m Migration) {
	if err := reg.Register(migrationID, m); err != nil {
		panic(err)
	}
}

// reg is a globally available register instance that must be used during the
// runtime to register migration functions.
// Register is declared as a separate type so that it can be tested without
// worrying about the global state.
var reg *register = newRegister()

func newRegister() *register {
	return &register{defs: make(map[string]Migration)}
}

type register struct {
	defs map[string]Migration
}

// Migration clubs together all requirements for executing a migration.
type Migration struct {
	// RequiredSigners is a collection of at least one Address. Transaction
	// must be signed by all specified address owners in order for the
	// migration to authorized for execution.
	// This is configured in code and not on the chain in order to avoid
	// chicken-egg problem.
	RequiredSigners []weave.Address

	// ChainIDs specifies which chains this migration can be executed on.
	// This is helpful because no two chains are the same and therefore now
	// two chains should require the same set of data migrations. Usually
	// we run staging/testing/production setup, each maintaining a
	// different state.
	ChainIDs []string

	Migrate func(context.Context, weave.KVStore) error
}

func (r *register) Register(migrationID string, m Migration) error {
	if _, ok := r.defs[migrationID]; ok {
		return errors.Wrapf(errors.ErrState, "migration %q already registered", migrationID)
	}

	switch n := len(migrationID); {
	case n < 6:
		return errors.Wrap(errors.ErrInput, "migration ID must be at least 6 characters long")
	case n > 128:
		return errors.Wrap(errors.ErrInput, "migration ID must be at most 128 characters long")
	}

	if len(m.RequiredSigners) == 0 {
		return errors.Wrap(errors.ErrEmpty, "at least one signer must be given")
	}
	for i, s := range m.RequiredSigners {
		if err := s.Validate(); err != nil {
			return errors.Wrapf(err, "required signer %d", i)
		}
	}

	r.defs[migrationID] = m
	return nil
}

func (r *register) Migration(id string) (*Migration, error) {
	m, ok := r.defs[id]
	if !ok {
		return nil, errors.Wrap(errors.ErrNotFound, "migration not declared")
	}
	return &m, nil
}
