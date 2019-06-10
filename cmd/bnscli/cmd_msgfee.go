package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/iov-one/weave/coin"
	"github.com/iov-one/weave/x/msgfee"
)

func msgfeeConf(nodeUrl string, msgPath string) (*coin.Coin, error) {
	queryUrl := nodeUrl + "/abci_query?path=%22/%22&data=%22msgfee:" + url.QueryEscape(msgPath) + "%22"
	resp, err := http.Get(queryUrl)
	if err != nil {
		return nil, fmt.Errorf("http request failed: %s", err)
	}
	defer resp.Body.Close()

	var payload struct {
		Result struct {
			Response struct {
				Value []byte
			}
		}
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("cannot decode payload: %s", err)
	}

	var fee msgfee.MsgFee
	if err := fee.Unmarshal(payload.Result.Response.Value); err != nil {
		return nil, fmt.Errorf("cannot decode model: %s", err)
	}
	return &fee.Fee, nil
}
