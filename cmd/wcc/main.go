package main

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/cmd/bnsd/app"
	"github.com/iov-one/weave/cmd/bnsd/client"
	"github.com/iov-one/weave/coin"
	"github.com/iov-one/weave/crypto"
	"github.com/iov-one/weave/x/cash"
	"github.com/iov-one/weave/x/gov"
)

var commands = map[string]func(input io.Reader, output io.Writer, args []string) error{
	"transfer-proposal": cmdNewTransferProposal,
	"sign":              cmdSignTransaction,
	"submit":            cmdSubmitTx,
}

func main() {
	if len(os.Args) == 1 {
		available := make([]string, 0, len(commands))
		for name := range commands {
			available = append(available, name)
		}
		fmt.Fprintf(os.Stderr, "Usage: %s <cmd> [<flags>]\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "\nAvailable commands are:\n\t%s\n", strings.Join(available, "\n\t"))
		os.Exit(2)
	}
	run, ok := commands[os.Args[1]]
	if !ok {
		available := make([]string, 0, len(commands))
		for name := range commands {
			available = append(available, name)
		}
		fmt.Fprintf(os.Stderr, "Unknown command %q\n", os.Args[1])
		fmt.Fprintf(os.Stderr, "\nAvailable commands are:\n\t%s\n", strings.Join(available, "\n\t"))
		os.Exit(2)
	}

	// Skip two first as second argument is the command name that we just
	// consumed.
	if err := run(os.Stdin, os.Stdout, os.Args[2:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func cmdNewTransferProposal(input io.Reader, output io.Writer, args []string) error {
	fl := flag.NewFlagSet("", flag.ExitOnError)
	fl.Usage = func() {
		fmt.Fprintln(flag.CommandLine.Output(), `
Transfer funds from source account to the destination.
		`)
		fl.PrintDefaults()
	}
	var (
		srcFl    = flAddress(fl, "src", "", "A source account address that the founds are send from.")
		dstFl    = flAddress(fl, "dst", "", "A destination account address that the founds are send to.")
		amountFl = flCoin(fl, "amount", "1 IOV", "An amount that is to be transferred between the source to the destination accounts.")
		memoFl   = fl.String("memo", "", "A short message attached to the transfer operation.")
		titleFl  = fl.String("title", "Transfer funds to distribution account", "The proposal title.")
		descFl   = fl.String("description", "Transfer funds to distribution account", "The proposal description.")
		eRuleFl  = fl.String("electionrule", "", "The ID of the election rule to be used.")
	)
	fl.Parse(args)

	option := app.ProposalOptions{
		Option: &app.ProposalOptions_SendMsg{
			SendMsg: &cash.SendMsg{
				Metadata: &weave.Metadata{Schema: 1},
				Src:      *srcFl,
				Dest:     *dstFl,
				Amount:   amountFl,
				Memo:     *memoFl,
				Ref:      nil,
			},
		},
	}
	rawOption, err := option.Marshal()
	if err != nil {
		return fmt.Errorf("cannot serialize %T option: %s", option, err)
	}
	tx := &app.Tx{
		Sum: &app.Tx_CreateProposalMsg{
			CreateProposalMsg: &gov.CreateProposalMsg{
				Metadata: &weave.Metadata{Schema: 1},
				Base: &gov.CreateProposalMsgBase{
					Title:          *titleFl,
					Description:    *descFl,
					StartTime:      weave.AsUnixTime(time.Now().Add(time.Minute)),
					ElectionRuleID: []byte(*eRuleFl),
				},
				RawOption: rawOption,
			},
		},
	}
	raw, err := tx.Marshal()
	if err != nil {
		return fmt.Errorf("cannot serialize transaction: %s", err)
	}
	_, err = output.Write(raw)
	return err
}

func flAddress(fl *flag.FlagSet, name, defaultVal, usage string) *weave.Address {
	var a weave.Address
	if defaultVal != "" {
		var err error
		a, err = weave.ParseAddress(defaultVal)
		if err != nil {
			panic(err.Error())
		}
	}
	fl.Var(&a, name, usage)
	return &a
}

func flCoin(fl *flag.FlagSet, name, defaultVal, usage string) *coin.Coin {
	var c coin.Coin
	if defaultVal != "" {
		var err error
		c, err = coin.ParseHumanFormat(defaultVal)
		if err != nil {
			panic(err.Error())
		}
	}
	fl.Var(&c, name, usage)
	return &c
}

func cmdSignTransaction(
	input io.Reader,
	output io.Writer,
	args []string,
) error {
	fl := flag.NewFlagSet("", flag.ExitOnError)
	fl.Usage = func() {
		fmt.Fprint(flag.CommandLine.Output(), `
Sign given transaction. This is decoding a transaction data from standard
input, adds a signature and writes back to standard output signed transaction
content.

`)
		fl.PrintDefaults()
	}
	var (
		tmAddrFl = fl.String("tm", "https://bns.NETWORK.iov.one:443", "Tendermint node address. Use proper NETWORK name.")
		keyFl    = fl.String("key", "", "Hex encoded, private key that transaction should be signed with.")
	)
	fl.Parse(args)

	if *keyFl == "" {
		return errors.New("private key is required")
	}
	key, err := decodePrivateKey(*keyFl)
	if err != nil {
		return fmt.Errorf("cannot decode private key: %s", err)
	}

	raw, err := ioutil.ReadAll(input)
	if err != nil {
		return fmt.Errorf("cannot read transaction: %s", err)
	}
	if len(raw) == 0 {
		return errors.New("no input data")
	}
	var tx app.Tx
	if err := tx.Unmarshal(raw); err != nil {
		return fmt.Errorf("cannot deserialize transaction: %s", err)
	}

	genesis, err := fetchGenesis(*tmAddrFl)
	if err != nil {
		return fmt.Errorf("cannot fetch genesis: %s", err)
	}

	bnsClient := client.NewClient(client.NewHTTPConnection(*tmAddrFl))
	aNonce := client.NewNonce(bnsClient, key.PublicKey().Address())
	if seq, err := aNonce.Next(); err != nil {
		return fmt.Errorf("cannot get the next sequence number: %s", err)
	} else {
		client.SignTx(&tx, key, genesis.ChainID, seq)
	}

	if raw, err := tx.Marshal(); err != nil {
		return fmt.Errorf("cannot serialize transaction: %s", err)
	} else {
		_, err = output.Write(raw)
	}
	return err
}

func decodePrivateKey(hexSeed string) (*crypto.PrivateKey, error) {
	data, err := hex.DecodeString(hexSeed)
	if err != nil {
		return nil, fmt.Errorf("cannot hex decode: %s", err)
	}
	if len(data) != 64 {
		return nil, errors.New("invalid key length")
	}
	key := &crypto.PrivateKey{
		Priv: &crypto.PrivateKey_Ed25519{Ed25519: data},
	}
	return key, nil
}

func fetchGenesis(serverURL string) (*genesis, error) {
	resp, err := http.Get(serverURL + "/genesis")
	if err != nil {
		return nil, fmt.Errorf("cannot fetch: %s", err)
	}
	defer resp.Body.Close()

	var payload struct {
		Result struct {
			Genesis genesis `json:"genesis"`
		} `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("cannot decode response: %s", err)
	}
	return &payload.Result.Genesis, nil
}

type genesis struct {
	ChainID string `json:"chain_id"`
}

func cmdSubmitTx(
	input io.Reader,
	output io.Writer,
	args []string,
) error {
	fl := flag.NewFlagSet("", flag.ExitOnError)
	fl.Usage = func() {
		fmt.Fprint(flag.CommandLine.Output(), `
Read binary serialized transaction from standard input and submit it.

Make sure to collect enough signatures before submitting the transaction.
`)
		fl.PrintDefaults()
	}
	var (
		tmAddrFl = fl.String("tm", "https://bns.NETWORK.iov.one:443", "Tendermint node address. Use proper NETWORK name.")
	)
	fl.Parse(args)

	raw, err := ioutil.ReadAll(input)
	if err != nil {
		return fmt.Errorf("cannot read transaction from input: %s", err)
	}
	if len(raw) == 0 {
		return errors.New("no input data")
	}
	var tx app.Tx
	if err := tx.Unmarshal(raw); err != nil {
		return fmt.Errorf("cannot deserialize transaction: %s", err)
	}
	bnsClient := client.NewClient(client.NewHTTPConnection(*tmAddrFl))

	if err := bnsClient.BroadcastTx(&tx).IsError(); err != nil {
		return fmt.Errorf("cannot broadcast transaction: %s", err)
	}
	return nil

}
