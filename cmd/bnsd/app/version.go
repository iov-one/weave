package bnsd

import "strconv"

// AppVersion should be set by build flags: `git describe --tags`
// It must be release tag without v prefix and dots
var AppVersion = "please set in makefile"

// tendermint expects app version as uint64
func getAppVersion() uint64 {
	appVersion, err := strconv.ParseUint(AppVersion, 10, 64)
	if err != nil {
		panic(err)
	}
	return appVersion
}
