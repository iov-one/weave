package gconf

import (
	"reflect"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/coin"
	"github.com/iov-one/weave/errors"
)

type Store interface {
	Get(key []byte) []byte
	Set(key, value []byte)
}

func Save(db Store, configuration interface{}) error {
	val := reflect.ValueOf(configuration)
	for val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	if val.Kind() != reflect.Struct {
		return errors.Wrapf(errors.ErrInvalidType, "configuration must be a structure, got %T", configuration)
	}
	tp := val.Type()

	for i := 0; i < tp.NumField(); i++ {
		field := val.Field(i)
		if !field.CanInterface() {
			continue
		}
		key := tp.PkgPath() + ":" + tp.Field(i).Name
		if err := saveValue(db, key, field.Interface()); err != nil {
			return errors.Wrapf(err, "cannot save %q field", key)
		}
	}
	return nil
}

func saveValue(db Store, key string, value interface{}) error {
	var cv ConfigurationValue

	// If supported, validate first to ensure we are not storing corrupted
	// data.
	type validator interface {
		Validate() error
	}
	if v, ok := value.(validator); ok {
		valueof := reflect.ValueOf(value)
		if valueof.Kind() != reflect.Ptr || !reflect.ValueOf(value).IsNil() {
			if err := v.Validate(); err != nil {
				return errors.Wrap(err, "validation failed")
			}
		}
	}

	switch v := value.(type) {
	case int64:
		cv.Value = &ConfigurationValue_Int64{Int64: v}
	case string:
		cv.Value = &ConfigurationValue_String_{String_: v}
	case weave.Address:
		cv.Value = &ConfigurationValue_Address{Address: v}
	case coin.Coin:
		cv.Value = &ConfigurationValue_Coin{Coin: &v}
	case *coin.Coin:
		cv.Value = &ConfigurationValue_Coin{Coin: v}
	default:
		return errors.Wrapf(errors.ErrInvalidType, "type %T is not supported", value)
	}

	raw, err := cv.Marshal()
	if err != nil {
		return errors.Wrap(err, "cannot marshal configuration value")
	}
	db.Set([]byte(key), raw)
	return nil
}

func Load(db Store, destination interface{}) error {
	val := reflect.ValueOf(destination)
	if val.Kind() != reflect.Ptr {
		return errors.Wrapf(errors.ErrInvalidType, "expected structure pointer, got %T", destination)
	}
	val = val.Elem()
	if val.Kind() != reflect.Struct {
		return errors.Wrapf(errors.ErrInvalidType, "expected structure pointer, got %T", destination)
	}
	if !val.CanSet() {
		return errors.Wrap(errors.ErrInvalidInput, "cannot set destination")
	}
	tp := val.Type()

	for i := 0; i < tp.NumField(); i++ {
		field := val.Field(i)
		if !field.CanSet() {
			continue
		}
		key := tp.PkgPath() + ":" + tp.Field(i).Name
		if err := loadValue(db, key, field.Addr().Interface()); err != nil {
			return errors.Wrapf(err, "cannot load %q field", key)
		}
	}
	return nil
}

func loadValue(db Store, key string, destination interface{}) error {
	destVal := reflect.ValueOf(destination)
	if destVal.Kind() != reflect.Ptr {
		return errors.Wrapf(errors.ErrInvalidType, "expected structure pointer, got %T", destination)
	}
	destVal = destVal.Elem()
	if !destVal.CanSet() {
		return errors.Wrap(errors.ErrInvalidInput, "cannot set destination")
	}

	raw := db.Get([]byte(key))
	if raw == nil {
		// If configuration does not exist then there is nothing to
		// set. This is not an error, because the first run of the
		// application will never have a configuration stored.
		return nil
	}

	var cv ConfigurationValue
	if err := cv.Unmarshal(raw); err != nil {
		return errors.Wrap(err, "cannot unmarshal configuration value")
	}
	switch confVal := cv.Value.(type) {
	case nil:
		destVal.Set(reflect.Zero(destVal.Type()))
	case *ConfigurationValue_Int64:
		destVal.SetInt(confVal.Int64)
	case *ConfigurationValue_String_:
		destVal.SetString(confVal.String_)
	case *ConfigurationValue_Address:
		destVal.SetBytes(confVal.Address)
	case *ConfigurationValue_Coin:
		if destVal.Kind() == reflect.Ptr {
			destVal.Set(reflect.ValueOf(confVal.Coin))
		} else {
			destVal.Set(reflect.ValueOf(*confVal.Coin))
		}
	default:
		return errors.Wrapf(errors.ErrInvalidType, "type %T not supported", cv.Value)
	}
	return nil
}
