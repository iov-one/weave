package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/iov-one/weave/cmd/bnsd/x/account"
	"github.com/iov-one/weave/cmd/bnsd/x/preregistration"
	"github.com/iov-one/weave/cmd/bnsd/x/qualityscore"
	"github.com/iov-one/weave/cmd/bnsd/x/termdeposit"
	"github.com/iov-one/weave/cmd/bnsd/x/username"
	"github.com/iov-one/weave/gconf"
	"github.com/iov-one/weave/migration"
	"github.com/iov-one/weave/x/cash"
	"github.com/iov-one/weave/x/msgfee"
	"github.com/iov-one/weave/x/txfee"
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
		Tendermint: env("TENDERMINT", "http://localhost:26657"),
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

	gconfConfigurations := map[string]func() gconf.Configuration{
		"account":         func() gconf.Configuration { return &account.Configuration{} },
		"cash":            func() gconf.Configuration { return &cash.Configuration{} },
		"migration":       func() gconf.Configuration { return &migration.Configuration{} },
		"msgfee":          func() gconf.Configuration { return &msgfee.Configuration{} },
		"preregistration": func() gconf.Configuration { return &preregistration.Configuration{} },
		"qualityscore":    func() gconf.Configuration { return &qualityscore.Configuration{} },
		"termdeposit":     func() gconf.Configuration { return &termdeposit.Configuration{} },
		"txfee":           func() gconf.Configuration { return &txfee.Configuration{} },
		"username":        func() gconf.Configuration { return &username.Configuration{} },
	}

	rt := http.NewServeMux()
	rt.Handle("/info", &InfoHandler{})
	rt.Handle("/blocks/", &BlocksHandler{bns: bnscli})
	rt.Handle("/account/domains", &AccountDomainsHandler{bns: bnscli})
	rt.Handle("/account/accounts", &AccountAccountsHandler{bns: bnscli})
	rt.Handle("/account/accounts/", &AccountAccountDetailHandler{bns: bnscli})
	rt.Handle("/termdeposit/contracts", &TermdepositContractsHandler{bns: bnscli})
	rt.Handle("/termdeposit/deposits", &TermdepositDepositsHandler{bns: bnscli})
	rt.Handle("/multisig/contracts", &MultisigContractsHandler{bns: bnscli})
	rt.Handle("/escrow/escrows", &EscrowEscrowsHandler{bns: bnscli})
	rt.Handle("/gov/proposals", &GovProposalsHandler{bns: bnscli})
	rt.Handle("/gov/votes", &GovVotesHandler{bns: bnscli})
	rt.Handle("/gconf/", &GconfHandler{bns: bnscli, confs: gconfConfigurations})
	rt.Handle("/", &DefaultHandler{})

	if err := http.ListenAndServe(conf.HTTP, rt); err != nil {
		return fmt.Errorf("http server: %s", err)
	}
	return nil
}
