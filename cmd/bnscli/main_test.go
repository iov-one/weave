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

// privKeyHex is a hex-encoded private key of an account with tokens on the test server
const privKeyHex = "d34c1970ae90acf3405f2d99dcaca16d0c7db379f4beafcfdf667b9d69ce350d27f5fb440509dfa79ec883a0510bc9a9614c3d44188881f0c5e402898b4bf3c9"

// addr is the hex address of the account that corresponds to privKeyHex
const addr = "E28AE9A6EB94FC88B73EB7CBD6B87BF93EB9BEF0"

func TestMain(m *testing.M) {
	code := runTestMain(m)
	os.Exit(code)
}

// we need to do setup in a separate function, so cleanup is properly called
// os.Exit(code) above will never call defer
func runTestMain(m *testing.M) int {
	var t mockAsserter

	home, cleanup := tmtest.SetupConfig(t, "testdata")
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	tmtest.RunBnsd(ctx, t, home)
	tmtest.RunTendermint(ctx, t, home)

	return m.Run()
}

// mockAsserter lets us use the assert calls even though we have testing.M not testing.T
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
	fmt.Println(args...)
}
func (mockAsserter) Logf(format string, args ...interface{}) {
	fmt.Printf(format, args...)
	fmt.Println("")
}
func (m mockAsserter) Skip(args ...interface{}) {
	m.Log(args...)
	os.Exit(0)
}
