package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
)

// buildVersion and buildHash are set by the build process.
var (
	buildHash    = "dev"
	buildVersion = "dev"
)

type configuration struct {
	HTTP       string
	Tendermint string
}

func main() {
	log.SetOutput(os.Stdout)
	log.SetFlags(log.LUTC | log.Lshortfile)
	log.SetPrefix(cutstr(buildHash, 6) + " ")

	conf := configuration{
		HTTP:       env("HTTP", ":8000"),
		Tendermint: env("TENDERMINT", "TODO"),
	}

	if err := run(conf); err != nil {
		log.Fatal(err)
	}
}

func cutstr(s string, maxchar int) string {
	if len(s) <= maxchar {
		return s
	}
	return s[:maxchar]
}

func env(name, fallback string) string {
	if v, ok := os.LookupEnv(name); ok {
		return v
	}
	return fallback
}

func run(conf configuration) error {
	bnscli := NewHTTPBnsClient(conf.Tendermint)

	rt := http.NewServeMux()
	rt.Handle("/info", &InfoHandler{})
	rt.Handle("/blocks/", &BlocksHandler{bns: bnscli})
	rt.Handle("/account/domains", &AccountDomainsHandler{bns: bnscli})
	rt.Handle("/account/accounts", &AccountAccountsHandler{bns: bnscli})
	rt.Handle("/account/accounts/", &AccountAccountDetailHandler{bns: bnscli})
	rt.Handle("/termdeposit/contracts", &TermdepositContractsHandler{bns: bnscli})
	rt.Handle("/termdeposit/deposits", &TermdepositDepositsHandler{bns: bnscli})
	rt.Handle("/", &DefaultHandler{})

	if err := http.ListenAndServe(conf.HTTP, rt); err != nil {
		return fmt.Errorf("http server: %s", err)
	}
	return nil
}
