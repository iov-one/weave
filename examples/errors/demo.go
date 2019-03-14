/*
package main demonstrates how we can print out our errors

meant for `go run .../demo.go`
*/
package main

import (
	"fmt"

	"github.com/iov-one/weave/errors"
)

func makeError() error {
	return errors.ErrNotFound.New("foo")
}

func otherError() error {
	return errors.ErrInvalidInput.New("unable to decode")
}

type foo struct {
	a int
}

func fullError() error {
	return errors.WithType(errors.ErrInvalidMsg, &foo{7})
}

func panicError() (err error) {
	defer errors.Recover(&err)
	panic("uh oh")
}

func show(err error) {
	fmt.Printf("Simple: %s\n", err)
	fmt.Printf("Verbose: %v\n", err)
	fmt.Printf("Full: %+v\n", err)
	fmt.Println("\n****")
}

func main() {
	show(makeError())
	show(otherError())
	show(fullError())
	show(panicError())
}
