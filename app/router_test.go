package app

import (
	"context"
	"testing"

	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/weavetest"
	"github.com/stretchr/testify/assert"
)

func TestRouter(t *testing.T) {
	r := NewRouter()
	const good, bad, missing = "good", "bad", "missing"

	// register some routers
	h := &weavetest.Handler{}
	r.Handle(good, h)
	r.Handle(bad, &weavetest.Handler{
		CheckErr:   errors.ErrHuman,
		DeliverErr: errors.ErrHuman,
	})

	// make sure invalid registrations panic
	assert.Panics(t, func() { r.Handle(good, h) })
	assert.Panics(t, func() { r.Handle("l:7", h) })

	// check proper paths work
	assert.Equal(t, 0, h.CallCount())
	_, err := r.Handler(good).Check(context.TODO(), nil, nil)
	assert.Nil(t, err)
	_, err = r.Handler(good).Deliver(context.TODO(), nil, nil)
	assert.Nil(t, err)
	// we count twice per decorator call
	assert.Equal(t, 2, h.CallCount())

	// check errors handler is also looked up
	_, err = r.Handler(bad).Deliver(context.TODO(), nil, nil)
	assert.Error(t, err)
	assert.False(t, errors.ErrNotFound.Is(err))
	assert.True(t, errors.ErrHuman.Is(err))
	assert.Equal(t, 2, h.CallCount())

	// make sure not found returns an error handler as well
	_, err = r.Handler(missing).Deliver(context.TODO(), nil, nil)
	assert.Error(t, err)
	assert.True(t, errors.ErrNotFound.Is(err))
	_, err = r.Handler(missing).Check(context.TODO(), nil, nil)
	assert.Error(t, err)
	assert.True(t, errors.ErrNotFound.Is(err))
	assert.Equal(t, 2, h.CallCount())
}
