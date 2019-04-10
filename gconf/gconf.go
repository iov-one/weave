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

func loadValue(db Store, key string, destination interface{}) error {
	value := reflect.ValueOf(destination)
	if value.Kind() != reflect.Ptr {
		return errors.Wrapf(errors.ErrInvalidType, "expected structure pointer, got %T", destination)
	}
	value = value.Elem()
	if value.Kind() != reflect.Struct {
		return errors.Wrapf(errors.ErrInvalidType, "expected structure pointer, got %T", destination)
	}

	raw := db.Get([]byte(key))
	if raw == nil {
		return errors.Wrap(errors.ErrNotFound, "configuration value not found")
	}

	panic("todo")
}
