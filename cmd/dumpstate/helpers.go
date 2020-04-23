package main

import (
	"os"
)

// env returns the value of an environment variable if provided (even if empty)
// or a fallback value.
func env(name, fallback string) string {
	if v, ok := os.LookupEnv(name); ok {
		return v
	}
	return fallback
}
