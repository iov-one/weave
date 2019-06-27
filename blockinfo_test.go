package weave_test

import (
	"os"
	"testing"
	"time"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/stretchr/testify/assert"
	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/libs/log"
)

func TestBlockInfo(t *testing.T) {
	blocktime, err := time.Parse(time.RFC3339, "2019-03-15T14:56:00Z")
	assert.Nil(t, err)
	unixtime := weave.AsUnixTime(blocktime)
	h := abci.Header{
		Height: 123,
		Time:   blocktime,
	}

	newLogger := log.NewTMLogger(os.Stdout)

	cases := map[string]struct {
		chainID      string
		logger       log.Logger
		err          *errors.Error
		expectLogger log.Logger
	}{
		"default logger": {
			chainID:      "test-chain",
			expectLogger: weave.DefaultLogger,
		},
		"custom logger": {
			chainID:      "test-chain",
			logger:       newLogger,
			expectLogger: newLogger,
		},
		"bad chain id": {
			chainID: "invalid;;chars",
			err:     errors.ErrInput,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			bi, err := weave.NewBlockInfo(h, weave.CommitInfo{}, tc.chainID, tc.logger)
			if tc.err != nil {
				if !tc.err.Is(err) {
					t.Fatalf("Unexpected error: %+v", err)
				}
				return
			}

			assert.Nil(t, err)
			assert.Equal(t, tc.expectLogger, bi.Logger())
			assert.Equal(t, int64(123), bi.Height())
			assert.Equal(t, blocktime, bi.BlockTime())
			assert.Equal(t, unixtime, bi.UnixTime())
		})
	}

}
