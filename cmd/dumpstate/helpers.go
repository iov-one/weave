package main

import (
	"fmt"
	"os"
)

// flagDie terminates the program when a flag parsing was not successful. This
// is a variable so that it can be overwritten for the tests.
var flagDie = func(description string, args ...interface{}) {
	s := fmt.Sprintf(description, args...)
	fmt.Fprintln(os.Stderr, s)
	os.Exit(2)
}

// env returns the value of an environment variable if provided (even if empty)
// or a fallback value.
func env(name, fallback string) string {
	if v, ok := os.LookupEnv(name); ok {
		return v
	}
	return fallback
}
