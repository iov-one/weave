package weave

import (
	"reflect"
	"testing"
)

func TestDedupe(t *testing.T) {
	specs := map[string]struct {
		Updates ValidatorUpdates
		Exp     ValidatorUpdates
		ExpZero ValidatorUpdates
	}{

		"Empty": {
			Updates: ValidatorUpdates{},
			Exp:     ValidatorUpdates{},
			ExpZero: ValidatorUpdates{},
		},
		"No Duplicates or zeroes": {
			Updates: ValidatorUpdates{
				ValidatorUpdates: []ValidatorUpdate{
					{Power: 1, PubKey: PubKey{Type: "123", Data: []byte("12")}},
					{Power: 3, PubKey: PubKey{Type: "123", Data: []byte("1234")}},
					{Power: 6, PubKey: PubKey{Type: "12", Data: []byte("1234")}},
				},
			},
			Exp: ValidatorUpdates{
				ValidatorUpdates: []ValidatorUpdate{
					{Power: 1, PubKey: PubKey{Type: "123", Data: []byte("12")}},
					{Power: 3, PubKey: PubKey{Type: "123", Data: []byte("1234")}},
					{Power: 6, PubKey: PubKey{Type: "12", Data: []byte("1234")}}},
			},
			ExpZero: ValidatorUpdates{
				ValidatorUpdates: []ValidatorUpdate{
					{Power: 1, PubKey: PubKey{Type: "123", Data: []byte("12")}},
					{Power: 3, PubKey: PubKey{Type: "123", Data: []byte("1234")}},
					{Power: 6, PubKey: PubKey{Type: "12", Data: []byte("1234")}}},
			},
		},
		"Duplicates and zeroes": {
			Updates: ValidatorUpdates{
				ValidatorUpdates: []ValidatorUpdate{
					{Power: 1, PubKey: PubKey{Type: "123", Data: []byte("12")}},
					{Power: 1, PubKey: PubKey{Type: "123", Data: []byte("12")}},
					{Power: 0, PubKey: PubKey{Type: "123", Data: []byte("1234")}},
					{Power: 6, PubKey: PubKey{Type: "12", Data: []byte("1234")}},
				},
			},
			Exp: ValidatorUpdates{
				ValidatorUpdates: []ValidatorUpdate{
					{Power: 1, PubKey: PubKey{Type: "123", Data: []byte("12")}},
					{Power: 0, PubKey: PubKey{Type: "123", Data: []byte("1234")}},
					{Power: 6, PubKey: PubKey{Type: "12", Data: []byte("1234")}},
				}},
			ExpZero: ValidatorUpdates{
				ValidatorUpdates: []ValidatorUpdate{
					{Power: 1, PubKey: PubKey{Type: "123", Data: []byte("12")}},
					{Power: 6, PubKey: PubKey{Type: "12", Data: []byte("1234")}},
				},
			},
		},
		"Zero duplicate": {
			Updates: ValidatorUpdates{
				ValidatorUpdates: []ValidatorUpdate{
					{Power: 1, PubKey: PubKey{Type: "123", Data: []byte("12")}},
					{Power: 0, PubKey: PubKey{Type: "123", Data: []byte("12")}},
					{Power: 6, PubKey: PubKey{Type: "12", Data: []byte("1234")}},
				},
			},
			Exp: ValidatorUpdates{
				ValidatorUpdates: []ValidatorUpdate{
					{Power: 0, PubKey: PubKey{Type: "123", Data: []byte("12")}},
					{Power: 6, PubKey: PubKey{Type: "12", Data: []byte("1234")}},
				}},
			ExpZero: ValidatorUpdates{
				ValidatorUpdates: []ValidatorUpdate{
					{Power: 6, PubKey: PubKey{Type: "12", Data: []byte("1234")}},
				},
			},
		},
	}
	for msg, spec := range specs {
		t.Run(msg, func(t *testing.T) {
			dedupe := spec.Updates.Deduplicate(false)
			if !reflect.DeepEqual(dedupe, spec.Exp) {
				t.Fatalf("expected %v to equal %+v", spec.Exp, dedupe)
			}

			dedupeZero := spec.Updates.Deduplicate(true)
			if !reflect.DeepEqual(dedupeZero, spec.ExpZero) {
				t.Fatalf("expected %v to equal %+v", spec.ExpZero, dedupeZero)
			}
		})
	}
}
