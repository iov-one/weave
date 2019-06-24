package errors

import (
	"reflect"
	"testing"
)

func TestFieldErrors(t *testing.T) {
	// Declare errors upfront so that DeepEqual can be used for comparison.
	var (
		unauthorizedNameErr = Field("name", ErrUnauthorized, "a")
		humanNameErr        = Field("name", ErrHuman, "b")
		emptyGenderErr      = Field("gender", ErrEmpty, "gender is required")
		userMultiErr        = Field("user", Append(
			humanNameErr,
			Append(emptyGenderErr, ErrDeleted),
		), "user data invalid")

		emptyGenderWrapErr = Field("gender", emptyGenderErr, "outer")
	)

	cases := map[string]struct {
		Err   error
		Field string
		Want  []error
	}{
		"a single error found by the name": {
			Err:   unauthorizedNameErr,
			Field: "name",
			Want:  []error{unauthorizedNameErr},
		},
		"two error found by the name": {
			Err: Append(
				unauthorizedNameErr,
				humanNameErr,
			),
			Field: "name",
			Want: []error{
				unauthorizedNameErr,
				humanNameErr,
			},
		},
		"field can contain a multierror": {
			Err:   userMultiErr,
			Field: "user",
			Want:  []error{userMultiErr},
		},
		"field can inspect errors tree to find match (name)": {
			Err:   userMultiErr,
			Field: "name",
			Want:  []error{humanNameErr},
		},
		"field can inspect errors tree to find match (gender)": {
			Err:   userMultiErr,
			Field: "gender",
			Want:  []error{emptyGenderErr},
		},
		"nil error returns nothing": {
			Err:   nil,
			Field: "foo",
			Want:  nil,
		},
		"error not found by the field name": {
			Err:   ErrUnauthorized,
			Field: "foo",
			Want:  nil,
		},
		"error not found by the wrong field name": {
			Err:   Field("a-name", ErrUnauthorized, "a description"),
			Field: "foo",
			Want:  nil,
		},
		"field is wrapped": {
			Err:   Wrap(Wrap(humanNameErr, "inner"), "outer"),
			Field: "name",
			Want:  []error{humanNameErr},
		},
		"multi error field is wrapped (gender)": {
			Err:   Wrap(Wrap(userMultiErr, "inner"), "outer"),
			Field: "gender",
			Want:  []error{emptyGenderErr},
		},
		"multi error field is wrapped (name)": {
			Err:   Wrap(Wrap(userMultiErr, "inner"), "outer"),
			Field: "name",
			Want:  []error{humanNameErr},
		},
		"multi error field is wrapped, no match": {
			Err:   Wrap(Wrap(userMultiErr, "inner"), "outer"),
			Field: "unknown-name",
			Want:  nil,
		},
		"multiple field wrap with most inner as the result": {
			Err:   Field("a", Field("b", humanNameErr, "b desc"), "a desc"),
			Field: "name",
			Want:  []error{humanNameErr},
		},
		"multiple field wrap with the same field return the most outside only": {
			Err:   emptyGenderWrapErr,
			Field: "gender",
			Want:  []error{emptyGenderWrapErr},
		},
		"complex error with multiple results": {
			Err: Wrap(Append(
				Wrap(unauthorizedNameErr, "a"),
				Wrap(humanNameErr, "b"),
				Wrap(emptyGenderErr, "c"),
			), "outer"),
			Field: "name",
			Want:  []error{unauthorizedNameErr, humanNameErr},
		},
	}

	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			got := FieldErrors(tc.Err, tc.Field)
			if !reflect.DeepEqual(tc.Want, got) {
				t.Logf("want: %#v", tc.Want)
				t.Logf(" got: %#v", got)
				t.Fatal("unexpected result")
			}
		})
	}
}
