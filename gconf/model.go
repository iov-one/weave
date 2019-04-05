package gconf

import (
	"reflect"
	"time"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/coin"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/orm"
)

var _ orm.CloneableData = (*Conf)(nil)

func (c *Conf) Validate() error {
	return nil
}

func (c *Conf) Copy() orm.CloneableData {
	// TODO
	return c
}

type ConfBucket struct {
	orm.Bucket
}

func NewConfBucket() *ConfBucket {
	return &ConfBucket{
		Bucket: orm.NewBucket("gconf", orm.NewSimpleObj(nil, &Conf{})),
	}
}

func NewConf(propName string, value interface{}) (orm.Object, error) {
	var conf Conf

	switch v := value.(type) {
	case string:
		conf.Value = &Conf_String_{String_: v}
	case int:
		// Support int type because this is what compiler type to any
		// number without explicit casting.
		conf.Value = &Conf_Int{Int: int64(v)}
	case int64:
		conf.Value = &Conf_Int{Int: v}
	case []byte:
		conf.Value = &Conf_Bytes{Bytes: v}
	case weave.Address:
		conf.Value = &Conf_Bytes{Bytes: v}
	case time.Duration:
		conf.Value = &Conf_Int{Int: int64(v)}
	case *coin.Coin:
		conf.Value = &Conf_Coin{Coin: v}
	case coin.Coin:
		conf.Value = &Conf_Coin{Coin: &v}
	default:
		return nil, errors.Wrapf(errors.ErrInvalidType, "%T", value)
	}

	return orm.NewSimpleObj([]byte(propName), &conf), nil
}

func (b *ConfBucket) Load(db weave.KVStore, propName string, dest interface{}) error {
	val := reflect.ValueOf(dest)
	if val.Kind() != reflect.Ptr {
		return errors.Wrap(errors.ErrInvalidInput, "destination must be a pointer")
	}
	val = val.Elem()
	if !val.CanSet() {
		return errors.Wrap(errors.ErrInvalidInput, "destination cannot be set")
	}

	obj, err := b.Get(db, []byte(propName))
	if err != nil {
		return errors.Wrap(err, "cannot get from bucket")
	}
	if obj == nil {
		return errors.ErrNotFound
	}
	c, ok := obj.Value().(*Conf)
	if !ok {
		return errors.Wrapf(errors.ErrInvalidType, "%T", obj.Value())
	}

	var setErr error
	func() {
		// It is less work and safer to handle panic than to program
		// all possible cases.
		defer func() {
			if r := recover(); r != nil {
				setErr = errors.Wrapf(errors.ErrInvalidInput, "cannot assign value: %s", r)
			}
		}()

		switch c := c.Value.(type) {
		case *Conf_Int:
			switch dest.(type) {
			case *time.Duration:
				val.Set(reflect.ValueOf(time.Duration(c.Int)))
			case *int:
				val.Set(reflect.ValueOf(int(c.Int)))
			default:
				val.Set(reflect.ValueOf(c.Int))
			}
		case *Conf_String_:
			val.Set(reflect.ValueOf(c.String_))
		case *Conf_Bytes:
			val.Set(reflect.ValueOf(c.Bytes))
		case *Conf_Coin:
			if val.Kind() == reflect.Ptr {
				val.Set(reflect.ValueOf(c.Coin))
			} else {
				val.Set(reflect.ValueOf(*c.Coin))
			}
		default:
			setErr = errors.Wrapf(errors.ErrInvalidType, "%T is not supported", c)
		}
	}()

	return setErr
}
