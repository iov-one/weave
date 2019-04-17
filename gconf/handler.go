package gconf

import (
	"reflect"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/x"
)

// OwnedConfig must have an Owner field in protobuf, which we can use to control who can update
type OwnedConfig interface {
	Unmarshaler
	ValidMarshaler
	GetOwner() weave.Address
}

type UpdateConfigurationHandler struct {
	pkg string
	// we require this type to load the data
	config OwnedConfig
	auth   x.Authenticator
}

var _ weave.Handler = UpdateConfigurationHandler{}

func NewUpdateConfigurationHandler(pkg string, config OwnedConfig, auth x.Authenticator) UpdateConfigurationHandler {
	return UpdateConfigurationHandler{
		pkg:    pkg,
		config: config,
		auth:   auth,
	}
}

func (h UpdateConfigurationHandler) Check(ctx weave.Context, store weave.KVStore, tx weave.Tx) (weave.CheckResult, error) {
	// TODO
	return weave.CheckResult{}, nil
}

// Deliver demos my concepts
func (h UpdateConfigurationHandler) Deliver(ctx weave.Context, store weave.KVStore, tx weave.Tx) (weave.DeliverResult, error) {
	// Load the current config from the store
	err := Load(store, h.pkg, h.config)
	if err != nil {
		return weave.DeliverResult{}, err
	}

	// make sure we are allowed to update
	owner := h.config.GetOwner()
	if owner == nil {
		return weave.DeliverResult{}, errors.Wrap(errors.ErrUnauthorized, "no owner can update config")
	}
	if !h.auth.HasAddress(ctx, owner) {
		return weave.DeliverResult{}, errors.Wrap(errors.ErrUnauthorized, "owner did not sign transaction")
	}

	// Get the payload from the Tx
	payload, err := h.getPayload(tx)
	if err != nil {
		return weave.DeliverResult{}, errors.Wrap(err, "cannot get message payload")
	}
	// patch current state
	err = h.patch(h.config, payload)
	if err != nil {
		return weave.DeliverResult{}, errors.Wrap(err, "cannot patch config with message payload")
	}
	// save to disk
	err = Save(store, h.pkg, h.config)
	if err != nil {
		return weave.DeliverResult{}, errors.Wrap(err, "cannot save updated config")
	}

	// success!!!
	return weave.DeliverResult{}, nil
}

// we are guaranteed that config and payload are the same type from getPayload
func (h UpdateConfigurationHandler) patch(config OwnedConfig, payload OwnedConfig) error {
	// now we got this, ensure same type as the h.config
	// TODO: test this
	pType := reflect.TypeOf(payload)
	cType := reflect.TypeOf(config)
	if !pType.ConvertibleTo(cType) {
		return errors.Wrap(errors.ErrInvalidMsg, "config in message doesn't match store")
	}

	cval := reflect.ValueOf(config).Elem()
	pval := reflect.ValueOf(payload).Elem()

	// TODO: do a patch here instead of overwriting... only non-zero fields
	cval.Set(pval)
	return nil
}

// getPayload expects the transaction to have a message with exactly one field (patch)
// it then ensures this field is the same as the config we have
func (h UpdateConfigurationHandler) getPayload(tx weave.Tx) (OwnedConfig, error) {
	msg, err := tx.GetMsg()
	if err != nil {
		return nil, err
	}

	// validate message
	if err := msg.Validate(); err != nil {
		return nil, err
	}

	// try to do (*Configuration).Patch and get the interface behind
	pval := reflect.ValueOf(msg)
	if pval.Kind() != reflect.Ptr || pval.Elem().Kind() != reflect.Struct {
		return nil, errors.Wrapf(errors.ErrInvalidInput, "invalid message container value: %T", msg)
	}
	val := pval.Elem()
	if val.NumField() != 1 {
		return nil, errors.Wrapf(errors.ErrInvalidInput, "unexpected message container field count: %d", val.NumField())
	}
	field := val.Field(0)
	if field.IsNil() {
		return nil, errors.Wrap(errors.ErrInvalidState, "payload is <nil>")
	}
	payload, ok := field.Interface().(OwnedConfig)
	if !ok {
		return nil, errors.Wrap(errors.ErrInvalidInput, "payload is not OwnedConfig")
	}

	return payload, nil
}
