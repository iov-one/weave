package weave

// Maj is the major version number (updated on breaking release)
const Maj = 0

// Min is the minor version number (updated on minor releases)
const Min = 4

// Fix is the patch number (updated on bugfix releases)
const Fix = 1

// Version should be set by build flags: `git describe --tags`
var Version = "please set in makefile"
