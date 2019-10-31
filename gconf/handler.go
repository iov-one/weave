package gconf

import (
	"reflect"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/x"
)

// OwnedConfig must have an Owner field in protobuf. A configuration update
// message must be signed by an owner in order to be authorized to apply the
// change.
type OwnedConfig interface {
	Unmarshaler
	ValidMarshaler
	GetOwner() weave.Address
}

type UpdateConfigurationHandler struct {
	pkg string
	// We require this type to load the data.
	config    OwnedConfig
	auth      x.Authenticator
	initAdmin func(weave.ReadOnlyKVStore) (weave.Address, error)
}

var _ weave.Handler = (*UpdateConfigurationHandler)(nil)

// NewUpdateConfigurationHandler returns a message handler that process
// configuration patch message.
//
// To pass authentication step, each message must be signed by the current
// configuration owner.
//
// A special chicken-egg problem appears when the configuration does not exist
// (it was not created via genesis). This is an issue, because without
// configuration we cannot configure configuration owner that can update the
// configuration. This means that the configuration cannot be created as well.
// A configuration is needed to create a configuration.
// To address the above issue, an optional `initConfAdmin` argument can be
// given to provide a creation only admin address. A good deafult is to use
// `migration.CurrentAdmin` function.
// `initConfAdmin` is used to authenticate the tranaction only when no
// configuration exist. Once a configuration is created, `initConfAdmin` is not
// used anymore and the autentication relies only on configuration's owner
// declaration.
func NewUpdateConfigurationHandler(
	pkg string,
	config OwnedConfig,
	auth x.Authenticator,
	initConfAdmin func(weave.ReadOnlyKVStore) (weave.Address, error),
) UpdateConfigurationHandler {
	return UpdateConfigurationHandler{
		pkg:       pkg,
		config:    config,
		auth:      auth,
		initAdmin: initConfAdmin,
	}
}

func (h UpdateConfigurationHandler) Check(ctx weave.Context, store weave.KVStore, tx weave.Tx) (*weave.CheckResult, error) {
	if err := h.applyTx(ctx, store, tx); err != nil {
		return nil, err
	}
	return &weave.CheckResult{}, nil
}

func (h UpdateConfigurationHandler) Deliver(ctx weave.Context, store weave.KVStore, tx weave.Tx) (*weave.DeliverResult, error) {
	if err := h.applyTx(ctx, store, tx); err != nil {
		return nil, err
	}
	return &weave.DeliverResult{}, nil
}

func (h UpdateConfigurationHandler) applyTx(ctx weave.Context, store weave.KVStore, tx weave.Tx) error {
	switch err := Load(store, h.pkg, h.config); {
	case err == nil:
		// Configuration owner must sign the transaction in order to
		// authenticate the change.
		owner := h.config.GetOwner()
		if owner == nil {
			return errors.Wrap(errors.ErrUnauthorized, "owner signature required")
		}
		if !h.auth.HasAddress(ctx, owner) {
			return errors.Wrap(errors.ErrUnauthorized, "owner did not sign transaction")
		}
	case errors.ErrNotFound.Is(err):
		// Configuration entity does not exist. It was not initialized
		// during via the genesis and will be created for the first
		// time now.
		if h.initAdmin == nil {
			return errors.Wrap(errors.ErrUnauthorized, "configuration does not exist and cannot be initialized")
		}
		admin, err := h.initAdmin(store)
		if err != nil {
			return errors.Wrap(err, "get init admin")
		}
		if !h.auth.HasAddress(ctx, admin) {
			return errors.Wrap(errors.ErrUnauthorized, "initialization admin signature required")
		}
	default:
		return errors.Wrap(err, "load current configuration")
	}

	payload, err := patchPayload(tx)
	if err != nil {
		return errors.Wrap(err, "cannot get message payload")
	}
	if err := patch(h.config, payload); err != nil {
		return errors.Wrap(err, "cannot patch config with message payload")
	}

	if err := Save(store, h.pkg, h.config); err != nil {
		return errors.Wrap(err, "cannot save updated config")
	}
	return nil
}

func patch(config OwnedConfig, payload OwnedConfig) error {
	// We are guaranteed that config and payload are the same type from
	// patchPayload.
	pType := reflect.TypeOf(payload)
	cType := reflect.TypeOf(config)
	if !pType.ConvertibleTo(cType) {
		return errors.Wrap(errors.ErrMsg, "config in message doesn't match store")
	}

	cval := reflect.ValueOf(config).Elem()
	pval := reflect.ValueOf(payload).Elem()

	for i := 0; i < cval.NumField(); i++ {
		got := pval.Field(i)

		// Zero values do not update the original configuration.
		if isZero(got) {
			continue
		}

		cval.Field(i).Set(got)
	}

	return nil
}

// isZero returns true if given value represents a zero value of a given type.
func isZero(val reflect.Value) bool {
	zero := reflect.Zero(val.Type()).Interface()
	return reflect.DeepEqual(val.Interface(), zero)
}

// patchPayload expects the transaction to have a message with "Patch" field of
// the same type as the configuration. Content of this field is extracted and
// returned.
func patchPayload(tx weave.Tx) (OwnedConfig, error) {
	msg, err := tx.GetMsg()
	if err != nil {
		return nil, err
	}

	// validate message
	if err := msg.Validate(); err != nil {
		return nil, err
	}

	// Try to do (*Configuration).Patch and get the interface behind.
	pval := reflect.ValueOf(msg)
	if pval.Kind() != reflect.Ptr || pval.Elem().Kind() != reflect.Struct {
		return nil, errors.Wrapf(errors.ErrInput, "invalid message container value: %T", msg)
	}
	val := pval.Elem()

	field := val.FieldByName("Patch")
	if field.IsNil() {
		return nil, errors.Wrap(errors.ErrState, `"Patch" field is required`)
	}
	payload, ok := field.Interface().(OwnedConfig)
	if !ok {
		return nil, errors.Wrap(errors.ErrInput, `"Patch" field is of a wrong type`)
	}
	return payload, nil
}
