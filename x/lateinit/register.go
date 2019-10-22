package lateinit

import (
	"context"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/gconf"
	"github.com/iov-one/weave/orm"
	"github.com/iov-one/weave/x"
)

// MustRegister registers an initialization function for given entity.
func MustRegister(
	initID string,
	requiredSigner weave.Address,
	entityID []byte,
	bucket orm.ModelBucket,
	entity orm.Model,
) {
	if err := reg.Register(initID, requiredSigner, entityID, bucket, entity); err != nil {
		panic(err)
	}
}

// MustRegisterGConfig is a specialization of MustRegister function meant to be
// used for registering of gconf configuration objects initialization.
func MustRegisterGConfig(
	pkgName string,
	requiredSigner weave.Address,
	entity orm.Model,
) {
	MustRegister(
		pkgName+"/configuration",
		requiredSigner,
		[]byte(pkgName),
		gconf.NewConfigurationModelBucket(),
		entity,
	)
}

// reg is a globally available register instance that must be used during the
// runtime to register initialization handlers.
// Register is declared as a separate type so that it can be tested without
// worrying about the global state.
var reg *register = newRegister()

func newRegister() *register {
	return &register{defs: make(map[string]createDef)}
}

type register struct {
	defs map[string]createDef
}

// createDef clubs together all parts required to initialize an entity.
type createDef struct {
	requiredSigner weave.Address
	entityID       []byte
	bucket         orm.ModelBucket
	entity         orm.Model
}

func (r *register) Register(
	initID string,
	requiredSigner weave.Address,
	entityID []byte,
	bucket orm.ModelBucket,
	entity orm.Model,
) error {
	if _, ok := r.defs[initID]; ok {
		return errors.Wrapf(errors.ErrState,
			"init function for entity %q and bucket %v already registered", entityID, bucket)
	}

	if err := entity.Validate(); err != nil {
		return errors.Wrap(err, "entity not valid")
	}

	switch n := len(initID); {
	case n < 6:
		return errors.Wrap(errors.ErrInput, "initialization ID must be at least 6 characters long")
	case n > 128:
		return errors.Wrap(errors.ErrInput, "initialization ID must be at most 128 characters long")
	}

	if err := requiredSigner.Validate(); err != nil {
		return errors.Wrap(err, "required signer")
	}

	r.defs[initID] = createDef{
		requiredSigner: requiredSigner,
		entityID:       entityID,
		bucket:         bucket,
		entity:         entity,
	}
	return nil
}

func (r *register) Exec(
	ctx context.Context,
	auth x.Authenticator,
	db weave.KVStore,
	initID string,
) error {
	def, ok := r.defs[initID]
	if !ok {
		return errors.Wrap(errors.ErrNotFound, "no initialization definition")
	}

	if !auth.HasAddress(ctx, def.requiredSigner) {
		return errors.ErrUnauthorized
	}

	switch err := def.bucket.Has(db, def.entityID); {
	case errors.ErrNotFound.Is(err):
		// All good.
	case err == nil:
		return errors.Wrapf(errors.ErrState, "entity %q already exists", def.entityID)
	default:
		return errors.Wrapf(err, "cannot check if entity %q exists", def.entityID)
	}

	if _, err := def.bucket.Put(db, def.entityID, def.entity); err != nil {
		return errors.Wrapf(err, "cannot save %q entity", def.entityID)
	}

	return nil
}

// RequiredSigner returns the address of a signer that signature is required in
// order to execute an initializatino instruction.
func (r *register) RequiredSigner(initID string) (weave.Address, error) {
	def, ok := r.defs[initID]
	if !ok {
		return nil, errors.Wrap(errors.ErrNotFound, "no initialization definition")
	}
	return def.requiredSigner, nil
}
