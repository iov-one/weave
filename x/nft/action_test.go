package nft

import "testing"

func TestIsValidAction(t *testing.T) {
	const customAction = "ThisIsAnActionRegisteredForTests"
	RegisterAction(customAction)

	cases := map[string]struct {
		action    interface{}
		wantValid bool
	}{
		"const declared action": {
			action:    UpdateDetails,
			wantValid: true,
		},
		"custom string action": {
			action:    string(customAction),
			wantValid: true,
		},
		"custom Action action": {
			action:    Action(customAction),
			wantValid: true,
		},
		"invalid action type": {
			action:    666,
			wantValid: false,
		},
		"action not registered": {
			action:    Action("NotRegisteredAction"),
			wantValid: false,
		},
	}

	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			if isValidAction(tc.action) != tc.wantValid {
				t.Fatalf("want valid=%v", tc.wantValid)
			}
		})
	}
}
