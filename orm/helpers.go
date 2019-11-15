package orm

import (
	"reflect"

	"github.com/iov-one/weave/errors"
)

type limitedIterator struct {
	remaining int
	// iterator is the underlying iterator
	iter SerialModelIterator
}

var _ SerialModelIterator = (*limitedIterator)(nil)

func (l *limitedIterator) LoadNext(dest SerialModel) error {
	if l.remaining > 0 {
		err := l.iter.LoadNext(dest)
		if err != nil {
			return err
		}
		l.remaining--
	}
	return errors.Wrap(errors.ErrIteratorDone, "iterator limit reached")
}

func (l *limitedIterator) Release() {
	l.iter.Release()
}

func LimitIterator(iter SerialModelIterator, limit int) SerialModelIterator {
	return &limitedIterator{iter: iter, remaining: limit}
}

func ToSlice(iter SerialModelIterator, destination SerialModelSlicePtr) error {
	dest := reflect.ValueOf(destination)
	if dest.Kind() != reflect.Ptr {
		return errors.Wrap(errors.ErrType, "destination must be a pointer to slice of SerialModels")
	}
	if dest.IsNil() {
		return errors.Wrap(errors.ErrImmutable, "got nil pointer")
	}
	dest = dest.Elem()
	if dest.Kind() != reflect.Slice {
		return errors.Wrap(errors.ErrType, "destination must be a pointer to slice of SerialModels")
	}

	var elemTyp reflect.Type
	if dest.Type().Elem().Kind() != reflect.Ptr {
		elemTyp = dest.Type().Elem()
	} else {
		elemTyp = dest.Type().Elem().Elem()
	}

	// Consume all the elements in iter to the destination
	for {
		d := reflect.New(elemTyp).Interface().(SerialModel)
		if err := iter.LoadNext(d); err != nil {
			if !errors.ErrIteratorDone.Is(err) {
				return err
			}
			// if iterator dones error received, means successfuly loaded iterator
			return nil
		}
		dest.Set(reflect.Append(dest, reflect.ValueOf(d)))
	}
}
