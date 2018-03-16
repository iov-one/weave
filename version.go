package weave

import "fmt"

// Maj is the major version number (updated on breaking release)
const Maj = 0
// Min is the minor version number (updated on minor releases)
const Min = 2
// Fix is the patch number (updated on bugfix releases)
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
