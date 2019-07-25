package main

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/iov-one/weave"
	bnsd "github.com/iov-one/weave/cmd/bnsd/app"
	"github.com/iov-one/weave/cmd/bnsd/client"
	"github.com/iov-one/weave/cmd/bnsd/x/username"
	"github.com/iov-one/weave/orm"
	"github.com/iov-one/weave/x/cash"
	"github.com/iov-one/weave/x/gov"
)

func cmdQuery(input io.Reader, output io.Writer, args []string) error {
	fl := flag.NewFlagSet("", flag.ExitOnError)
	fl.Usage = func() {
		fmt.Fprint(flag.CommandLine.Output(), `
Execute a ABCI query and print JSON encoded result.
`)
		fl.PrintDefaults()
	}
	var (
		tmAddrFl = fl.String("tm", env("BNSCLI_TM_ADDR", "https://bns.NETWORK.iov.one:443"),
			"Tendermint node address. Use proper NETWORK name. You can use BNSCLI_TM_ADDR environment variable to set it.")
		pathFl        = fl.String("path", "", "Path to be queried. Must be one of the supported.")
		dataFl        = fl.String("data", "", "individual query data. Format depends on the queried entity. Use 'id/version' for electoraterules, electorates")
		prefixQueryFl = fl.Bool("prefix", false, "If true, use prefix queries instead of the exact match with provided data.")
	)
	fl.Parse(args)

	conf, ok := queries[*pathFl]
	if !ok {
		var paths []string
		for p := range queries {
			paths = append(paths, p)
		}
		return fmt.Errorf("available query paths:\n\t- %s", strings.Join(paths, "\n\t- "))
	}

	var data []byte
	if len(*dataFl) != 0 {
		var err error
		if data, err = conf.encID(*dataFl); err != nil {
			return fmt.Errorf("can not encode data: %s", err)
		}
	}
	queryPath := *pathFl
	if *prefixQueryFl || *dataFl == "" {
		queryPath += "?" + weave.PrefixQueryMod
	}

	bnsClient := client.NewClient(client.NewHTTPConnection(*tmAddrFl))
	resp, err := bnsClient.AbciQuery(queryPath, data)
	if err != nil {
		return fmt.Errorf("failed to run query: %s", err)
	}

	result := make([]keyval, 0, len(resp.Models))
	for i, m := range resp.Models {
		obj := conf.newObj()
		if err := obj.Unmarshal(m.Value); err != nil {
			return fmt.Errorf("failed to unmarshal model %d: %s", i, err)
		}
		key, err := conf.decKey(m.Key)
		if err != nil {
			return fmt.Errorf("cannot decode %x key: %s", m.Key, err)
		}
		result = append(result, keyval{Key: key, Value: obj})
	}
	pretty, err := json.MarshalIndent(result, "", "\t")
	if err != nil {
		return fmt.Errorf("cannot JSON serialize: %s", err)
	}
	_, err = output.Write(pretty)
	return err
}

type keyval struct {
	Key   string
	Value model
}

// queries contains a mapping of query path to that query specifics. Each query
// returns a custom model type and may use different ID encoding pattern.
var queries = map[string]struct {
	// newObj returns a new instance of the model that the result of the
	// ABCI query should be extracted into.
	newObj func() model
	// decKey is used to decode key value returned by the ABCI query and
	// transform it into human readable form.
	decKey func([]byte) (string, error)
	// encID is used to parse input format of the ID and encode it into
	// form that will be passed to the ABCI query. The format can differ
	// from decKey if we use secondary index for matching.
	encID func(string) ([]byte, error)
}{
	"/proposals": {
		newObj: func() model { return &extendedProposal{} },
		decKey: sequenceKey,
		encID:  numericID,
	},
	"/proposals/author": {
		newObj: func() model { return &extendedProposal{} },
		decKey: sequenceKey,
		encID:  addressID,
	},
	"/proposals/electorate": {
		newObj: func() model { return &extendedProposal{} },
		decKey: sequenceKey,
		encID:  numericID,
	},
	"/electionrules": {
		newObj: func() model { return &gov.ElectionRule{} },
		decKey: refKey,
		encID:  refID,
	},
	"/electorates": {
		newObj: func() model { return &gov.Electorate{} },
		decKey: refKey,
		encID:  refID,
	},
	"/electorates/elector": {
		newObj: func() model { return &gov.Electorate{} },
		decKey: refKey,
		encID:  addressID,
	},
	"/votes": {
		newObj: func() model { return &gov.Vote{} },
		decKey: rawKey,
		encID:  addressID,
	},
	"/votes/proposals": {
		newObj: func() model { return &gov.Vote{} },
		decKey: rawKey,
		encID:  numericID,
	},
	"/votes/electors": {
		newObj: func() model { return &gov.Vote{} },
		decKey: rawKey,
		encID:  addressID,
	},
	"/usernames": {
		newObj: func() model { return &username.Token{} },
		decKey: rawKey,
		encID:  addressID,
	},
	"/usernames/owner": {
		newObj: func() model { return &username.Token{} },
		decKey: rawKey,
		encID:  addressID,
	},
	"/wallets": {
		newObj: func() model { return &cash.Set{} },
		decKey: rawKey,
		encID:  addressID,
	},
}

// model is an entity used by weave to store data. This interface is
// implemented by any protobuf message.
type model interface {
	Unmarshal([]byte) error
}

// refID expects `id/version` pair with integers.
func refID(s string) ([]byte, error) {
	tokens := strings.Split(s, "/")

	var version uint32
	switch len(tokens) {
	case 1:
		// Allow providing just the ID value to enable prefix queries.
		// This is a special case.
	case 2:
		if n, err := strconv.ParseUint(tokens[1], 10, 32); err != nil {
			return nil, fmt.Errorf("cannot decode version: %s", err)
		} else {
			version = uint32(n)
		}
	default:
		return nil, errors.New("invalid ID format, use 'id/version'")
	}

	encID := make([]byte, 8)
	if n, err := strconv.ParseUint(tokens[0], 10, 64); err != nil {
		return nil, fmt.Errorf("cannot decode ID: %s", err)
	} else {
		binary.BigEndian.PutUint64(encID, n)
	}

	ref := orm.VersionedIDRef{ID: encID, Version: version}
	return ref.Marshal()
}

func addressID(s string) ([]byte, error) {
	return weave.ParseAddress(s)
}

func refKey(raw []byte) (string, error) {
	// Skip the prefix, being the characters before : (including separator)
	val := raw[bytes.Index(raw, []byte(":"))+1:]

	var ref orm.VersionedIDRef
	if err := ref.Unmarshal(val); err != nil {
		return "", fmt.Errorf("cannot unmarshal versioned key: %s", err)
	}
	id := binary.BigEndian.Uint64(ref.ID)
	return fmt.Sprintf("%d/%d", id, ref.Version), nil
}

func numericID(s string) ([]byte, error) {
	n, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("cannot parse number: %s", err)
	}
	encID := make([]byte, 8)
	binary.BigEndian.PutUint64(encID, n)
	return encID, nil
}

func sequenceKey(raw []byte) (string, error) {
	// Skip the prefix, being the characters before : (including separator)
	seq := raw[bytes.Index(raw, []byte(":"))+1:]
	if len(seq) != 8 {
		return "", fmt.Errorf("invalid sequence length: %d", len(seq))
	}
	n := binary.BigEndian.Uint64(seq)
	return fmt.Sprint(int64(n)), nil
}

func rawKey(raw []byte) (string, error) {
	return hex.EncodeToString(raw), nil
}

// extendedProposal is the gov.Proposal with an additional field to extract
// RawOption. When serialized using JSON, this structure produce the same
// result as the gov.Proposal with an addition of an attribute representing
// deserialized (human readable) form of the message that is proposed.
type extendedProposal struct {
	gov.Proposal
	// Option contains a deserialized value of the RawOption
	Option interface{} `json:"executed_when_accepted"`
}

// Unmarshal implements protobuf unmarshaler interface.
func (p *extendedProposal) Unmarshal(raw []byte) error {
	if err := p.Proposal.Unmarshal(raw); err != nil {
		return fmt.Errorf("cannot unmarshal proposal: %s", err)
	}
	var opts bnsd.ProposalOptions
	if err := opts.Unmarshal(p.Proposal.RawOption); err != nil {
		return fmt.Errorf("cannot unmarshal proposal option: %s", err)
	}
	p.Option = opts.GetOption()
	return nil
}
