package nft

import "regexp"

const (
	minIDLength = 4
	maxIDLength = 256
)

var (
	isValidAction = regexp.MustCompile(`^[A-Z_]{4,32}$`).MatchString
)

type Validation struct {
}

func (*Validation) IsValidAction(action string) bool {
	return isValidAction(action)
}

func (*Validation) IsValidTokenID(id []byte) bool {
	return len(id) >= minIDLength && len(id) <= maxIDLength
}
