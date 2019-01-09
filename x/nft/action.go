package nft

import (
	"fmt"
	"regexp"
	"sync"
)

// Action represents available and supported by the implementation actions.
// This is just a string type alias, but using it increase the clarity of the
// API.
type Action string

// nft package provides default set of actions.
const (
	UpdateDetails   Action = "ActionUpdateDetails"
	Transfer        Action = "ActionTransfer"
	UpdateApprovals Action = "ActionUpdateApprovals"
)

// RegisterAction introduce an Action to the extension.
//
// Every action must be registered before being used. This is a mandatory step
// so that the validation process can regcognise known actions. Registration is
// global.
// Registration should be done during the program initialization phase. Failed
// registration result in panic.
func RegisterAction(actions ...Action) {
	validActions.Lock()
	defer validActions.Unlock()

	for _, a := range actions {
		if !validActionString(string(a)) {
			panic(fmt.Sprintf("invalid action name: %s", a))
		}
		validActions.set[a] = struct{}{}
	}
}

// Because we allow clients to register any action string, we must ensure that
// certain convention is preserved.
var validActionString = regexp.MustCompile(`^[A-Za-z]{4,32}$`).MatchString

var validActions = struct {
	sync.RWMutex
	set map[Action]struct{}
}{
	set: map[Action]struct{}{
		// Actions for which support is implemented in nft.
		UpdateDetails:   struct{}{},
		Transfer:        struct{}{},
		UpdateApprovals: struct{}{},
	},
}

// isValidAction returns true if given value is a valid action name. Action can
// be of type string or Action.
func isValidAction(action interface{}) bool {
	var name Action

	switch a := action.(type) {
	case Action:
		name = a
	case string:
		name = Action(a)
	default:
		return false
	}

	validActions.RLock()
	_, ok := validActions.set[name]
	validActions.RUnlock()
	return ok
}
