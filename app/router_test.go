package app

import (
	"context"
	"testing"

	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/weavetest"
	"github.com/iov-one/weave/weavetest/assert"
)

func TestRouterSuccess(t *testing.T) {
	r := NewRouter()

	var (
		msg     = &weavetest.Msg{RoutePath: "test/good"}
		handler = &weavetest.Handler{}
	)

	r.Handle(msg, handler)

	if _, err := r.Check(context.TODO(), nil, &weavetest.Tx{Msg: msg}); err != nil {
		t.Fatalf("check failed: %s", err)
	}
	if _, err := r.Deliver(context.TODO(), nil, &weavetest.Tx{Msg: msg}); err != nil {
		t.Fatalf("delivery failed: %s", err)
	}
	assert.Equal(t, 2, handler.CallCount())
}

func TestRouterNoHandler(t *testing.T) {
	r := NewRouter()

	tx := &weavetest.Tx{Msg: &weavetest.Msg{RoutePath: "test/secret"}}

	if _, err := r.Check(context.TODO(), nil, tx); !errors.ErrNotFound.Is(err) {
		t.Fatalf("expected not found error, got %s", err)
	}
	if _, err := r.Deliver(context.TODO(), nil, tx); !errors.ErrNotFound.Is(err) {
		t.Fatalf("expected not found error, got %s", err)
	}
}

func TestRegisteringInvalidMessagePath(t *testing.T) {
	r := NewRouter()
	assert.Panics(t, func() {
		r.Handle(&weavetest.Msg{RoutePath: ": "}, &weavetest.Handler{})
	})
}

func TestRegisteringMessageHandlerTwice(t *testing.T) {
	r := NewRouter()
	r.Handle(&weavetest.Msg{RoutePath: "test/msg"}, &weavetest.Handler{})
	assert.Panics(t, func() {
		r.Handle(&weavetest.Msg{RoutePath: "test/msg"}, &weavetest.Handler{})
	})
}
