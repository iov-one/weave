package app

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/iov-one/weave/x"
)

func TestRouter(t *testing.T) {
	var help x.TestHelpers

	r := NewRouter()
	good, bad, missing := "good", "bad", "missing"
	msg := "foo"

	// register some routers
	counter := help.CountingHandler()
	r.Handle(good, counter)
	r.Handle(bad, help.ErrorHandler(fmt.Errorf("foo")))

	// make sure invalid registrations panic
	assert.Panics(t, func() { r.Handle(good, counter) })
	assert.Panics(t, func() { r.Handle("l:7", counter) })

	// check proper paths work
	assert.Equal(t, 0, counter.GetCount())
	_, err := r.Handler(good).Check(nil, nil, nil)
	assert.NoError(t, err)
	_, err = r.Handler(good).Deliver(nil, nil, nil)
	assert.NoError(t, err)
	// we count twice per decorator call
	assert.Equal(t, 2, counter.GetCount())

	// check errors handler is also looked up
	_, err = r.Handler(bad).Deliver(nil, nil, nil)
	assert.Error(t, err)
	assert.False(t, IsNoSuchPathErr(err))
	assert.Equal(t, msg, err.Error())
	assert.Equal(t, 2, counter.GetCount())

	// make sure not found returns an error handler as well
	_, err = r.Handler(missing).Deliver(nil, nil, nil)
	assert.Error(t, err)
	assert.True(t, IsNoSuchPathErr(err))
	_, err = r.Handler(missing).Check(nil, nil, nil)
	assert.Error(t, err)
	assert.True(t, IsNoSuchPathErr(err))
	assert.Equal(t, 2, counter.GetCount())
}
