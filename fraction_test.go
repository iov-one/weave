package weave

import (
	"bytes"
	"encoding/json"
	"reflect"
	"testing"
)

func TestFractionUnmarshalJSON(t *testing.T) {
	cases := map[string]struct {
		raw      string
		wantFrac Fraction
		wantErr  bool
	}{
		"zero": {
			raw:      `"0"`,
			wantFrac: Fraction{Denominator: 1},
			wantErr:  false,
		},
		"integer human format number": {
			raw:      `"4"`,
			wantFrac: Fraction{Numerator: 4, Denominator: 1},
			wantErr:  false,
		},
		"zero numerator": {
			raw:      `"0/123"`,
			wantFrac: Fraction{Denominator: 123},
			wantErr:  false,
		},
		"zero numerator and denominator": {
			raw:      `"0/1"`,
			wantFrac: Fraction{Denominator: 1},
			wantErr:  false,
		},
		"human readable format": {
			raw:      `"1/2"`,
			wantFrac: Fraction{Numerator: 1, Denominator: 2},
			wantErr:  false,
		},
		"human readable format, too many separators": {
			raw:     `"1/2/3"`,
			wantErr: true,
		},
		"human readable format, floating point number": {
			raw:     `"1/3.3"`,
			wantErr: true,
		},
		"human readable format, signed number": {
			raw:     `"-1"`,
			wantErr: true,
		},
		"verbose format": {
			raw:      `{"numerator": 1, "denominator": 2}`,
			wantFrac: Fraction{Numerator: 1, Denominator: 2},
			wantErr:  false,
		},
		"verbose format only denominator": {
			raw:      `{"denominator": 2}`,
			wantFrac: Fraction{Numerator: 0, Denominator: 2},
			wantErr:  false,
		},
		"verbose format only numerator": {
			raw:      `{"numerator": 2}`,
			wantFrac: Fraction{Numerator: 2, Denominator: 0},
			wantErr:  false,
		},
		"random string characters": {
			raw:     `"asdlkhsdalhksda"`,
			wantErr: true,
		},
		"number is not acceptable": {
			raw:     `12345`,
			wantErr: true,
		},
		"whitespace is irrelevant for human format": {
			raw:      `"\t 3 / \t 2 "`,
			wantFrac: Fraction{Numerator: 3, Denominator: 2},
			wantErr:  false,
		},
	}

	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			var got Fraction
			err := json.Unmarshal([]byte(tc.raw), &got)
			gotErr := err != nil
			if tc.wantErr != gotErr {
				t.Fatalf("want error=%v, got %v", tc.wantErr, err)
			}
			if err == nil && !reflect.DeepEqual(got, tc.wantFrac) {
				t.Fatalf("want %+v, got %+v", tc.wantFrac, got)
			}
		})
	}
}

func TestFractionMarshalJSON(t *testing.T) {
	f := Fraction{Numerator: 4, Denominator: 5}
	b, err := json.Marshal(f)
	if err != nil {
		t.Fatal(err)
	}
	const want = `{"numerator":4,"denominator":5}`
	if !bytes.Equal(b, []byte(want)) {
		t.Fatalf("unexpected JSON format: %q", b)
	}
}

func TestFractionCompare(t *testing.T) {
	cases := map[string]struct {
		a, b Fraction
		Want int
	}{
		"two equal values": {
			a:    Fraction{Numerator: 1, Denominator: 2},
			b:    Fraction{Numerator: 2, Denominator: 4},
			Want: 0,
		},
		"a greater than b": {
			a:    Fraction{Numerator: 3, Denominator: 5},
			b:    Fraction{Numerator: 2, Denominator: 4},
			Want: 1,
		},
		"b greater than a": {
			a:    Fraction{Numerator: 3, Denominator: 5},
			b:    Fraction{Numerator: 3, Denominator: 4},
			Want: -1,
		},
		"a is zero": {
			a:    Fraction{Numerator: 0, Denominator: 2},
			b:    Fraction{Numerator: 3, Denominator: 4},
			Want: -1,
		},
		"a is zero value": {
			a:    Fraction{Numerator: 0, Denominator: 0},
			b:    Fraction{Numerator: 3, Denominator: 4},
			Want: -1,
		},
		"b is zero": {
			a:    Fraction{Numerator: 1, Denominator: 2},
			b:    Fraction{Numerator: 0, Denominator: 3},
			Want: 1,
		},
		"b is zero value": {
			a:    Fraction{Numerator: 1, Denominator: 2},
			b:    Fraction{Numerator: 0, Denominator: 0},
			Want: 1,
		},
		"a and b are zero": {
			a:    Fraction{Numerator: 0, Denominator: 123},
			b:    Fraction{Numerator: 0, Denominator: 0},
			Want: 0,
		},
	}

	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			got := tc.a.Compare(tc.b)
			if got != tc.Want {
				t.Logf("%v compare %v", tc.a, tc.b)
				t.Fatalf("want %d, got %d", tc.Want, got)
			}
		})
	}
}
