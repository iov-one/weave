package weavetest

import (
	"reflect"
	"testing"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
)

func TestHandlerWithError(t *testing.T) {
	h := Handler{
		CheckErr:   errors.ErrUnauthorized,
		DeliverErr: errors.ErrNotFound,
	}

	_, err := h.Check(nil, nil, nil)
	if want := errors.ErrUnauthorized; !want.Is(err) {
		t.Errorf("want %q, got %q", want, err)
	}

	_, err = h.Deliver(nil, nil, nil)
	if want := errors.ErrNotFound; !want.Is(err) {
		t.Errorf("want %q, got %q", want, err)
	}
}

//nolint
func TestHandlerCallCount(t *testing.T) {
	var h Handler

	assertHCounts(t, &h, 0, 0)

	h.Check(nil, nil, nil)
	assertHCounts(t, &h, 1, 0)

	h.Check(nil, nil, nil)
	assertHCounts(t, &h, 2, 0)

	h.Deliver(nil, nil, nil)
	assertHCounts(t, &h, 2, 1)

	h.Deliver(nil, nil, nil)
	assertHCounts(t, &h, 2, 2)

	// Failing counter must increment as well.
	h.CheckErr = errors.ErrNotFound
	h.DeliverErr = errors.ErrNotFound

	h.Check(nil, nil, nil)
	assertHCounts(t, &h, 3, 2)

	h.Deliver(nil, nil, nil)
	assertHCounts(t, &h, 3, 3)
}

func assertHCounts(t *testing.T, h *Handler, wantCheck, wantDeliver int) {
	t.Helper()
	if got := h.CheckCallCount(); got != wantCheck {
		t.Errorf("want %d checks, got %d", wantCheck, got)
	}
	if got := h.DeliverCallCount(); got != wantDeliver {
		t.Errorf("want %d delivers, got %d", wantDeliver, got)
	}
	wantTotal := wantCheck + wantDeliver
	if got := h.CallCount(); got != wantTotal {
		t.Errorf("want %d total, got %d", wantTotal, got)
	}
}

func TestHandlerResult(t *testing.T) {
	wantCres := weave.CheckResult{
		Data:         []byte("foo"),
		GasAllocated: 5,
	}
	wantDres := weave.DeliverResult{
		Data:    []byte("bar"),
		GasUsed: 824,
	}
	h := Handler{
		CheckResult:   wantCres,
		DeliverResult: wantDres,
	}

	gotCres, _ := h.Check(nil, nil, nil)
	if !reflect.DeepEqual(&wantCres, gotCres) {
		t.Fatalf("got check result: %+v", gotCres)
	}
	gotDres, _ := h.Deliver(nil, nil, nil)
	if !reflect.DeepEqual(&wantDres, gotDres) {
		t.Fatalf("got deliver result: %+v", gotDres)
	}
}
