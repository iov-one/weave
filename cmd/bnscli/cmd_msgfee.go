package main

import (
	"github.com/iov-one/weave/coin"
	"github.com/iov-one/weave/x/msgfee"
)

func msgfeeConf(nodeUrl string, msgPath string) (*coin.Coin, error) {
	store := tendermintStore(nodeUrl)
	b := msgfee.NewMsgFeeBucket()
	return b.MessageFee(store, msgPath)
}
