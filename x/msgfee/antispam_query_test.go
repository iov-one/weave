package msgfee

import (
	"testing"

	"github.com/iov-one/weave/coin"
	"github.com/iov-one/weave/weavetest/assert"
)

func TestAntiSpamQuery(t *testing.T) {
	initialCoin := coin.Coin{
		Whole:      1,
		Fractional: 3,
		Ticker:     "DOGE",
	}

	q := NewAntiSpamQuery(initialCoin)
	model, err := q.Query(nil, "", nil)
	assert.Nil(t, err)

	modelNum := len(model)

	if modelNum != 1 {
		t.Fatalf("expected 1 model, got %d", modelNum)
	}

	c := coin.Coin{}
	err = c.Unmarshal(model[0].Value)
	assert.Nil(t, err)

	if !c.Equals(initialCoin) {
		t.Fatalf("expected coin %s to equal %s", c.String(), initialCoin.String())
	}

}
