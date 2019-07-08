package main

import (
	"encoding/binary"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/cmd/bnsd/client"
	"github.com/iov-one/weave/orm"
	"github.com/iov-one/weave/x/gov"
)

type respDecoder interface {
	Unmarshal([]byte) error
}
type idEncoder func(string) ([]byte, error)

var resultParser = map[string]func() respDecoder{
	"/proposal":      func() respDecoder { return &gov.Proposal{} },
	"/electionRules": func() respDecoder { return &gov.ElectionRule{} },
	"/electorates":   func() respDecoder { return &gov.Electorate{} },
	"/vote":          func() respDecoder { return &gov.Vote{} },
}
var idEncoders = map[string]idEncoder{
	"/proposal": func(s string) (bytes []byte, e error) {
		x, err := strconv.ParseUint(s, 10, 64)
		if err != nil {
			return nil, err
		}
		encID := make([]byte, 8)
		binary.BigEndian.PutUint64(encID, x)
		return encID, nil
	},
	"/electionRules": idRefDecoder,
	"/electorates":   idRefDecoder,
	//"/vote": func() respDecoder { return &gov.Vote{} },
}

func cmdQuery(input io.Reader, output io.Writer, args []string) error {
	fl := flag.NewFlagSet("", flag.ExitOnError)
	fl.Usage = func() {
		fmt.Fprintln(flag.CommandLine.Output(), `
Abci Query `)
		fl.PrintDefaults()
	}
	var (
		tmAddrFl = fl.String("tm", env("BNSCLI_TM_ADDR", "https://bns.NETWORK.iov.one:443"),
			"Tendermint node address. Use proper NETWORK name. You can use BNSCLI_TM_ADDR environment variable to set it.")
		pathFl        = fl.String("path", "", "query path")
		dataFl        = fl.String("data", "", "individual query data: id,version for electorateRules, electorates")
		prefixQueryFl = fl.Bool("prefix-mode", false, "optional parameter to enable prefix queries")
	)
	fl.Parse(args)
	if len(*pathFl) == 0 {
		flagDie("non empty path required")
	}

	bnsClient := client.NewClient(client.NewHTTPConnection(*tmAddrFl))
	var data []byte
	if len(*dataFl) != 0 {
		var err error
		decoder, ok := idEncoders[*pathFl]
		if !ok {
			return fmt.Errorf("no id decoder for path %q", *pathFl)
		}
		if data, err = decoder(*dataFl); err != nil {
			return fmt.Errorf("can not encode data: %s", err)
		}
	}
	queryPath := *pathFl
	if *prefixQueryFl {
		queryPath += "?" + weave.PrefixQueryMod
	}
	resp, err := bnsClient.AbciQuery(queryPath, data)
	if err != nil {
		return fmt.Errorf("failed to run query: %s", err)
	}

	p, ok := resultParser[*pathFl]
	if !ok {
		return fmt.Errorf("no decoder for path %q", *pathFl)
	}
	result := make([]interface{}, 0, len(resp.Models))
	for i, m := range resp.Models {
		obj := p()
		if err := obj.Unmarshal(m.Value); err != nil {
			return fmt.Errorf("failed to unmarshal model %d: %s", i, err)
		}
		result = append(result, obj)
	}
	pretty, err := json.MarshalIndent(result, "", "\t")
	if err != nil {
		return fmt.Errorf("cannot JSON serialize: %s", err)
	}
	_, err = output.Write(pretty)

	return err
}

// idRefDecoder expects `id/version` pair with integers
func idRefDecoder(s string) ([]byte, error) {
	tokens := strings.Split(s, ",")
	var version uint32
	if len(tokens) == 2 {
		x, err := strconv.ParseUint(tokens[1], 10, 32)
		if err != nil {
			return nil, err
		}
		version = uint32(x)
	}
	x, err := strconv.ParseUint(tokens[0], 10, 64)
	if err != nil {
		return nil, err
	}
	encID := make([]byte, 8)
	binary.BigEndian.PutUint64(encID, x)
	ref := &orm.VersionedIDRef{ID: encID, Version: version}
	return ref.Marshal()
}
