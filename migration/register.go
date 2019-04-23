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

// NoModyfication is a migration function that migrates data that requires no
// change. It should be used to register migrations that do not require any
// modyfications.
func NoModyfication(ctx weave.Context, db weave.KVStore, msgOrModel Payload) error {
	return nil
}

func newRegister() *register {
	return &register{
		handlers: make(map[payloadVersion]Migrator),
	}
}

type register struct {
	handlers map[payloadVersion]Migrator
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
	tp := reflect.TypeOf(msgOrModel)
	for tp.Kind() == reflect.Ptr {
		tp = tp.Elem()
	}
	if tp.Kind() != reflect.Struct {
		return errors.Wrapf(errors.ErrInvalidInput, "only struct can be migrated, got %T", msgOrModel)
	}

	pv := payloadVersion{
		version: migrationTo,
		payload: tp,
	}
	if _, ok := r.handlers[pv]; ok {
		return errors.Wrapf(errors.ErrDuplicate, "already registered: %s.%s:%d", tp.PkgPath(), tp.Name(), migrationTo)
	}
	r.handlers[pv] = fn
	return nil
}

func (r *register) Apply(ctx weave.Context, db weave.KVStore, msgOrModel Payload, migrateTo uint32) error {
	tp := reflect.TypeOf(msgOrModel)
	for tp.Kind() == reflect.Ptr {
		tp = tp.Elem()
	}
	if tp.Kind() != reflect.Struct {
		return errors.Wrapf(errors.ErrInvalidInput, "only struct can be migrated, got %T", msgOrModel)
	}

	header := msgOrModel.GetHeader()
	if header == nil {
		return errors.Wrap(errors.ErrInvalidState, "nil payload header")
	}
	for v := header.Schema; v <= migrateTo; v++ {
		migrate, ok := r.handlers[payloadVersion{payload: tp, version: v}]
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
// Validation method is called only on the final version of the payload.
func Apply(ctx weave.Context, db weave.KVStore, msgOrModel Payload, migrateTo uint32) error {
	return reg.Apply(ctx, db, msgOrModel, migrateTo)
}
