package weave

import "fmt"

const Maj = "0"
const Min = "1"
const Fix = "0"

// add a suffix (-dev, -alpha, -beta, etc) if not tagged release
const Suffix = "-dev"

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
