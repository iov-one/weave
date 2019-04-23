package migration

import (
	"reflect"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
)

// Payload is implemented by both weave.Msg and models used by the orm package.
// Schema migration supports both of those data types.
type Payload interface {
	GetHeader() *weave.Header
	Validate() error
}

// Migrator is a function that migrates a data entity from version
// requiredVersion-1 to requested version.
type Migrator func(ctx weave.Context, db weave.KVStore, msgOrModel Payload) error

// NoModification is a migration function that migrates data that requires no
// change. It should be used to register migrations that do not require any
// modyfications.
func NoModification(ctx weave.Context, db weave.KVStore, msgOrModel Payload) error {
	return nil
}

func newRegister() *register {
	return &register{
		migrateTo: make(map[payloadVersion]Migrator),
	}
}

type register struct {
	migrateTo map[payloadVersion]Migrator
}

// payloadVersion references a message or a model at a given schema version.
type payloadVersion struct {
	payload reflect.Type
	version uint32
}

func (r *register) MustRegister(migrationTo uint32, msgOrModel Payload, fn Migrator) {
	if err := r.Register(migrationTo, msgOrModel, fn); err != nil {
		panic(err)
	}
}

func (r *register) Register(migrationTo uint32, msgOrModel Payload, fn Migrator) error {
	if migrationTo < 1 {
		return errors.Wrap(errors.ErrInvalidInput, "minimal allowed version is 1")
	}

	tp := reflect.TypeOf(msgOrModel)

	if migrationTo > 1 {
		prev := payloadVersion{
			version: migrationTo - 1,
			payload: tp,
		}
		if _, ok := r.migrateTo[prev]; !ok {
			return errors.Wrapf(errors.ErrInvalidInput, "missing %d version migration", prev.version)
		}
	}

	pv := payloadVersion{
		version: migrationTo,
		payload: tp,
	}
	if _, ok := r.migrateTo[pv]; ok {
		return errors.Wrapf(errors.ErrDuplicate,
			"already registered: %s.%s:%d", tp.PkgPath(), tp.Name(), migrationTo)
	}
	r.migrateTo[pv] = fn
	return nil
}

func (r *register) Apply(ctx weave.Context, db weave.KVStore, msgOrModel Payload, migrateTo uint32) error {
	if migrateTo < 1 {
		return errors.Wrap(errors.ErrInvalidInput, "minimal allowed version is 1")
	}

	header := msgOrModel.GetHeader()
	if header == nil {
		return errors.Wrap(errors.ErrInvalidState, "nil payload header")
	}
	if header.Schema < 1 {
		return errors.Wrap(errors.ErrInvalidState, "header schema version below 1")
	}

	tp := reflect.TypeOf(msgOrModel)
	for v := header.Schema + 1; v <= migrateTo; v++ {
		migrate, ok := r.migrateTo[payloadVersion{payload: tp, version: v}]
		if !ok {
			return errors.Wrapf(errors.ErrInvalidState, "migration to version %d missing", v)
		}
		if err := migrate(ctx, db, msgOrModel); err != nil {
			return errors.Wrapf(err, "migration to version %d", v)
		}
		header.Schema = v
	}

	if err := msgOrModel.Validate(); err != nil {
		return errors.Wrap(err, "validation")
	}
	return nil
}

// reg is a globally available register instance that must be used during the
// runtime to register migration handlers.
// Register is declared as a separate type so that it can be tested without
// worrying about the global state.
var reg *register = newRegister()

// MustRegister registers a migration function for a given message or model.
// Migration function will be called when migrating data from a version one
// less than migrationTo value.
// Minimal allowed migrationTo version is 1. Version upgrades for each type
// must be registered in sequentional order.
func MustRegister(migrationTo uint32, msgOrModel Payload, fn Migrator) {
	reg.MustRegister(migrationTo, msgOrModel, fn)
}

// Apply updates a payload by applying all missing data migrations. Even a no
// modyfication migration is updating the header to point to the latest data
// format version.
//
// Because changes are applied directly on the passed payload, even if this
// function fails some of the data migrations might be applied.
//
// A valid message header must contain a schema version greater than zero. Not
// migrated message (initial state) is always having a header schema value set
// to 1.
//
// Validation method is called only on the final version of the payload.
func Apply(ctx weave.Context, db weave.KVStore, msgOrModel Payload, migrateTo uint32) error {
	return reg.Apply(ctx, db, msgOrModel, migrateTo)
}
