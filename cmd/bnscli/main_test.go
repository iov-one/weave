package main

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/iov-one/weave/tmtest"
)

// taken from testdata/config/config.toml - rpc.laddr
const tmURL = "http://localhost:44444"

func TestMain(m *testing.M) {
	var t mockAsserter

	home, cleanup := tmtest.SetupConfig(t, "testdata")
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	defer tmtest.RunBnsd(ctx, t, home)()
	defer tmtest.RunTendermint(ctx, t, home)()

	code := m.Run()
	os.Exit(code)
}

type mockAsserter struct{}

var _ tmtest.TestReporter = mockAsserter{}

func (mockAsserter) Helper() {}
func (mockAsserter) Fatal(args ...interface{}) {
	msg := fmt.Sprint(args...)
	panic(msg)
}
func (mockAsserter) Fatalf(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	panic(msg)
}
func (mockAsserter) Log(args ...interface{}) {
	fmt.Print(args...)
}
func (mockAsserter) Logf(format string, args ...interface{}) {
	fmt.Printf(format, args...)
}
func (m mockAsserter) Skip(args ...interface{}) {
	m.Log(args...)
	os.Exit(0)
}
