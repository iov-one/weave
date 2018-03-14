package weave

import "fmt"

// Note: update VersionTest when changing the version
const Maj = 0
const Min = 2
const Fix = 0

// Suffix used when not a tagged release (eg. -dev, -alpha, -beta, etc)
const Suffix = ""

// version is private to avoid modifications
var version = fmt.Sprintf("v%d.%d.%d%s", Maj, Min, Fix, Suffix)

// GitCommit set by build flags
var GitCommit = ""

// Version is the string to be displayed
func Version() string {
	v := version
	if GitCommit != "" {
		v += " " + GitCommit
	}
	return v
}
