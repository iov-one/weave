package weavetest

import (
	"testing"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
)

func TestSuccessfulDecorator(t *testing.T) {
	var (
		d Decorator
		h Handler
	)

	_, _ = d.Check(nil, nil, nil, &h)
	assertHCounts(t, &h, 1, 0)

	_, _ = d.Deliver(nil, nil, nil, &h)
	assertHCounts(t, &h, 1, 1)
}

func TestDecoratorWithError(t *testing.T) {
	d := Decorator{
		CheckErr:   errors.ErrUnauthorized,
		DeliverErr: errors.ErrNotFound,
	}

	// When using an error returning decorator, handler is never called.
	// Otherwise using nil would panic.
	var handler weave.Handler = nil

	_, err := d.Check(nil, nil, nil, handler)
	if want := errors.ErrUnauthorized; !want.Is(err) {
		t.Errorf("want %q, got %q", want, err)
	}

	_, err = d.Deliver(nil, nil, nil, handler)
	if want := errors.ErrNotFound; !want.Is(err) {
		t.Errorf("want %q, got %q", want, err)
	}
}

//nolint
func TestDecoratorCallCount(t *testing.T) {
	var d Decorator

	assertDCounts(t, &d, 0, 0)

	d.Check(nil, nil, nil, &Handler{})
	assertDCounts(t, &d, 1, 0)

	d.Check(nil, nil, nil, &Handler{})
	assertDCounts(t, &d, 2, 0)

	d.Deliver(nil, nil, nil, &Handler{})
	assertDCounts(t, &d, 2, 1)

	d.Deliver(nil, nil, nil, &Handler{})
	assertDCounts(t, &d, 2, 2)

	// Failing counter must increment as well.
	d.CheckErr = errors.ErrNotFound
	d.DeliverErr = errors.ErrNotFound

	d.Check(nil, nil, nil, &Handler{})
	assertDCounts(t, &d, 3, 2)

	d.Deliver(nil, nil, nil, &Handler{})
	assertDCounts(t, &d, 3, 3)
}

func assertDCounts(t *testing.T, d *Decorator, wantCheck, wantDeliver int) {
	t.Helper()
	if got := d.CheckCallCount(); got != wantCheck {
		t.Errorf("want %d checks, got %d", wantCheck, got)
	}
	if got := d.DeliverCallCount(); got != wantDeliver {
		t.Errorf("want %d delivers, got %d", wantDeliver, got)
	}
	wantTotal := wantCheck + wantDeliver
	if got := d.CallCount(); got != wantTotal {
		t.Errorf("want %d total, got %d", wantTotal, got)
	}
}
