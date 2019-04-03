package msgfee

import (
	"testing"

	"github.com/iov-one/weave/coin"
)

func TestAntiSpamQuery(t *testing.T) {
	initialCoin := coin.Coin{
		Whole:      0,
		Fractional: 0,
		Ticker:     "DOGE",
	}

	q := NewAntiSpamQuery(initialCoin)
	model, err := q.Query(nil, "", nil)
	if err != nil {
		t.Fatalf("unexpected error: %s ", err.Error())
	}

	modelNum := len(model)

	if modelNum != 1 {
		t.Fatalf("expected 1 model, got %d", modelNum)
	}

	c := coin.Coin{}
	err = c.Unmarshal(model[0].Value)
	if err != nil {
		t.Fatalf("unexpected unmarshaling error: %s ", err.Error())
	}

	if !(c).Equals(initialCoin) {
		t.Fatalf("expected coin %s to equal %s", c.String(), initialCoin.String())
	}

}
