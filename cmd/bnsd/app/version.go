package bnsd

import "strconv"

// AppVersion should be set by release tag without v prefix and dots, example: v1.0.1 -> 101
var AppVersion = "0"

// tendermint expects app version as uint64
func getAppVersion() uint64 {
	appVersion, err := strconv.ParseUint(AppVersion, 10, 64)
	if err != nil {
		panic(err)
	}
	return appVersion
}
