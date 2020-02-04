package handlers

import (
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/iov-one/weave/cmd/bnsapi/client"
	"github.com/iov-one/weave/cmd/bnsapi/util"
	"log"
	"math"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/cmd/bnsd/x/account"
	"github.com/iov-one/weave/cmd/bnsd/x/termdeposit"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/gconf"
	"github.com/iov-one/weave/orm"
	"github.com/iov-one/weave/x/escrow"
	"github.com/iov-one/weave/x/gov"
	"github.com/iov-one/weave/x/multisig"
)

type GovProposalsHandler struct {
	Bns client.BnsClient
}

func (h *GovProposalsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	if !atMostOne(q, "author", "electorate", "electorate_id") {
		JSONErr(w, http.StatusBadRequest, "At most one filter can be used at a time.")
		return
	}

	var it client.ABCIIterator
	offset := extractIDFromKey(q.Get("offset"))
	if e := q.Get("electorate"); len(e) > 0 {
		rawAddr, err := base64.StdEncoding.DecodeString(e)
		if err != nil {
			JSONErr(w, http.StatusBadRequest, "electorate address must be a base64 encoded value.")
			return
		}
		end := nextKeyValue(rawAddr)
		it = client.ABCIRangeQuery(r.Context(), h.Bns, "/proposals/electorate", fmt.Sprintf("%x:%x:%x", rawAddr, offset, end))
	} else if e := q.Get("electorate_id"); len(e) > 0 {
		n, err := strconv.ParseInt(e, 10, 64)
		if err != nil {
			JSONErr(w, http.StatusBadGateway, "electorate_id must be an integer contract sequence number.")
			return
		}
		start := encodeSequence(uint64(n))
		end := nextKeyValue(start)
		it = client.ABCIRangeQuery(r.Context(), h.Bns, "/proposals/electorate", fmt.Sprintf("%x:%x:%x", start, offset, end))
	} else if s := q.Get("author"); len(s) > 0 {
		rawAddr, err := weave.ParseAddress(s)
		if err != nil {
			JSONErr(w, http.StatusBadRequest, "author address must be a valid address value.")
			return
		}
		end := nextKeyValue(rawAddr)
		it = client.ABCIRangeQuery(r.Context(), h.Bns, "/proposals/author", fmt.Sprintf("%x:%x:%x", rawAddr, offset, end))
	} else {
		it = client.ABCIRangeQuery(r.Context(), h.Bns, "/proposals", fmt.Sprintf("%x:", offset))
	}

	objects := make([]KeyValue, 0, paginationMaxItems)
fetchProposals:
	for {
		var p gov.Proposal
		switch key, err := it.Next(&p); {
		case err == nil:
			objects = append(objects, KeyValue{
				Key:   key,
				Value: &p,
			})
			if len(objects) == paginationMaxItems {
				break fetchProposals
			}
		case errors.ErrIteratorDone.Is(err):
			break fetchProposals
		default:
			log.Printf("gov proposals ABCI query: %s", err)
			JSONErr(w, http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))
			return
		}
	}

	JSONResp(w, http.StatusOK, struct {
		Objects []KeyValue `json:"objects"`
	}{
		Objects: objects,
	})
}

type GovVotesHandler struct {
	Bns client.BnsClient
}

func (h *GovVotesHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	if !atMostOne(q, "proposal", "proposal_id", "elector", "elector_id") {
		JSONErr(w, http.StatusBadRequest, "At most one filter can be used at a time.")
		return
	}

	var it client.ABCIIterator
	offset := extractIDFromKey(q.Get("offset"))
	if e := q.Get("elector"); len(e) > 0 {
		rawAddr, err := weave.ParseAddress(e)
		if err != nil {
			JSONErr(w, http.StatusBadRequest, "elector ID address must be a valid address value..")
			return
		}
		end := nextKeyValue(rawAddr)
		it = client.ABCIRangeQuery(r.Context(), h.Bns, "/votes/electors", fmt.Sprintf("%x:%x:%x", rawAddr, offset, end))
	} else if e := q.Get("elector_id"); len(e) > 0 {
		// TODO - is elector the same as electorate?
		n, err := strconv.ParseInt(e, 10, 64)
		if err != nil {
			JSONErr(w, http.StatusBadGateway, "elector_id must be an integer contract sequence number.")
			return
		}
		start := encodeSequence(uint64(n))
		end := nextKeyValue(start)
		it = client.ABCIRangeQuery(r.Context(), h.Bns, "/votes/electors", fmt.Sprintf("%x:%x:%x", start, offset, end))
	} else if p := q.Get("proposal"); len(p) > 0 {
		rawAddr, err := weave.ParseAddress(p)
		if err != nil {
			JSONErr(w, http.StatusBadRequest, "proposal ID address must be a valid address value..")
			return
		}
		end := nextKeyValue(rawAddr)
		it = client.ABCIRangeQuery(r.Context(), h.Bns, "/votes/proposals", fmt.Sprintf("%x:%x:%x", rawAddr, offset, end))
	} else if p := q.Get("proposal_id"); len(p) > 0 {
		n, err := strconv.ParseInt(p, 10, 64)
		if err != nil {
			JSONErr(w, http.StatusBadGateway, "proposal_id must be an integer contract sequence number.")
			return
		}
		start := encodeSequence(uint64(n))
		end := nextKeyValue(start)
		it = client.ABCIRangeQuery(r.Context(), h.Bns, "/votes/proposals", fmt.Sprintf("%x:%x:%x", start, offset, end))
	} else {
		it = client.ABCIRangeQuery(r.Context(), h.Bns, "/votes", fmt.Sprintf("%x:", offset))
	}

	objects := make([]KeyValue, 0, paginationMaxItems)
fetchVotes:
	for {
		var v gov.Vote
		switch key, err := it.Next(&v); {
		case err == nil:
			objects = append(objects, KeyValue{
				Key:   key,
				Value: &v,
			})
			if len(objects) == paginationMaxItems {
				break fetchVotes
			}
		case errors.ErrIteratorDone.Is(err):
			break fetchVotes
		default:
			log.Printf("gov votes ABCI query: %s", err)
			JSONErr(w, http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))
			return
		}
	}

	JSONResp(w, http.StatusOK, struct {
		Objects []KeyValue `json:"objects"`
	}{
		Objects: objects,
	})
}

type EscrowEscrowsHandler struct {
	Bns client.BnsClient
}

func (h *EscrowEscrowsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	if !atMostOne(q, "source", "destination") {
		JSONErr(w, http.StatusBadRequest, "At most one filter can be used at a time.")
		return
	}

	var it client.ABCIIterator
	offset := extractIDFromKey(q.Get("offset"))
	if d := q.Get("destination"); len(d) > 0 {
		rawAddr, err := weave.ParseAddress(d)
		if err != nil {
			JSONErr(w, http.StatusBadRequest, "Destination address must be a valid address value..")
			return
		}
		end := nextKeyValue(rawAddr)
		it = client.ABCIRangeQuery(r.Context(), h.Bns, "/escrows/destination", fmt.Sprintf("%x:%x:%x", rawAddr, offset, end))
	} else if s := q.Get("source"); len(s) > 0 {
		rawAddr, err := weave.ParseAddress(s)
		if err != nil {
			JSONErr(w, http.StatusBadRequest, "Source address must be a valid address value..")
			return
		}
		end := nextKeyValue(rawAddr)
		it = client.ABCIRangeQuery(r.Context(), h.Bns, "/escrows/source", fmt.Sprintf("%x:%x:%x", rawAddr, offset, end))
	} else {
		it = client.ABCIRangeQuery(r.Context(), h.Bns, "/escrows", fmt.Sprintf("%x:", offset))
	}

	objects := make([]KeyValue, 0, paginationMaxItems)
fetchEscrows:
	for {
		var e escrow.Escrow
		switch key, err := it.Next(&e); {
		case err == nil:
			objects = append(objects, KeyValue{
				Key:   key,
				Value: &e,
			})
			if len(objects) == paginationMaxItems {
				break fetchEscrows
			}
		case errors.ErrIteratorDone.Is(err):
			break fetchEscrows
		default:
			log.Printf("escrow ABCI query: %s", err)
			JSONErr(w, http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))
			return
		}
	}

	JSONResp(w, http.StatusOK, struct {
		Objects []KeyValue `json:"objects"`
	}{
		Objects: objects,
	})
}

type MultisigContractsHandler struct {
	Bns client.BnsClient
}

func (h *MultisigContractsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	offset := extractIDFromKey(r.URL.Query().Get("offset"))
	it := client.ABCIRangeQuery(r.Context(), h.Bns, "/contracts", fmt.Sprintf("%x:", offset))

	objects := make([]KeyValue, 0, paginationMaxItems)
fetchContracts:
	for {
		var c multisig.Contract
		switch key, err := it.Next(&c); {
		case err == nil:
			objects = append(objects, KeyValue{
				Key:   key,
				Value: &c,
			})
			if len(objects) == paginationMaxItems {
				break fetchContracts
			}
		case errors.ErrIteratorDone.Is(err):
			break fetchContracts
		default:
			log.Printf("multisig contract ABCI query: %s", err)
			JSONErr(w, http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))
			return
		}
	}

	JSONResp(w, http.StatusOK, struct {
		Objects []KeyValue `json:"objects"`
	}{
		Objects: objects,
	})
}

type TermdepositContractsHandler struct {
	Bns client.BnsClient
}

func (h *TermdepositContractsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	offset := extractIDFromKey(r.URL.Query().Get("offset"))
	it := client.ABCIRangeQuery(r.Context(), h.Bns, "/depositcontracts", fmt.Sprintf("%x:", offset))

	objects := make([]KeyValue, 0, paginationMaxItems)
fetchContracts:
	for {
		var c termdeposit.DepositContract
		switch key, err := it.Next(&c); {
		case err == nil:
			objects = append(objects, KeyValue{
				Key:   key,
				Value: &c,
			})
			if len(objects) == paginationMaxItems {
				break fetchContracts
			}
		case errors.ErrIteratorDone.Is(err):
			break fetchContracts
		default:
			log.Printf("termdeposit contract ABCI query: %s", err)
			JSONErr(w, http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))
			return
		}
	}

	JSONResp(w, http.StatusOK, struct {
		Objects []KeyValue `json:"objects"`
	}{
		Objects: objects,
	})
}

type TermdepositDepositsHandler struct {
	Bns client.BnsClient
}

func (h *TermdepositDepositsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	if !atMostOne(q, "depositor", "contract_id", "contract") {
		JSONErr(w, http.StatusBadRequest, "At most one filter can be used at a time.")
		return
	}

	var it client.ABCIIterator
	offset := extractIDFromKey(q.Get("offset"))
	if d := q.Get("depositor"); len(d) > 0 {
		rawAddr, err := weave.ParseAddress(d)
		if err != nil {
			JSONErr(w, http.StatusBadRequest, "Depositor address must be a valid address value..")
			return
		}
		end := nextKeyValue(rawAddr)
		it = client.ABCIRangeQuery(r.Context(), h.Bns, "/deposits/depositor", fmt.Sprintf("%s:%x:%x", d, offset, end))
	} else if c := q.Get("contract_id"); len(c) > 0 {
		n, err := strconv.ParseInt(c, 10, 64)
		if err != nil {
			JSONErr(w, http.StatusBadGateway, "contract_id must be an integer contract sequence number.")
			return
		}
		cid := encodeSequence(uint64(n))
		end := nextKeyValue(cid)
		it = client.ABCIRangeQuery(r.Context(), h.Bns, "/deposits/contract", fmt.Sprintf("%x:%x:%x", cid, offset, end))
	} else if c := q.Get("contract"); len(c) > 0 {
		cid, err := base64.StdEncoding.DecodeString(c)
		if err != nil {
			JSONErr(w, http.StatusBadGateway, "Contract must be a base64 encoded contract key.")
			return
		}
		end := nextKeyValue(cid)
		it = client.ABCIRangeQuery(r.Context(), h.Bns, "/deposits/contract", fmt.Sprintf("%x:%x:%x", cid, offset, end))
	} else {
		it = client.ABCIRangeQuery(r.Context(), h.Bns, "/deposits", fmt.Sprintf("%x:", offset))
	}

	objects := make([]KeyValue, 0, paginationMaxItems)
fetchDeposits:
	for {
		var d termdeposit.Deposit
		switch key, err := it.Next(&d); {
		case err == nil:
			objects = append(objects, KeyValue{
				Key:   key,
				Value: &d,
			})
			if len(objects) == paginationMaxItems {
				break fetchDeposits
			}
		case errors.ErrIteratorDone.Is(err):
			break fetchDeposits
		default:
			log.Printf("termdeposit deposit ABCI query: %s", err)
			JSONErr(w, http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))
			return
		}
	}

	JSONResp(w, http.StatusOK, struct {
		Objects []KeyValue `json:"objects"`
	}{
		Objects: objects,
	})
}

type GconfHandler struct {
	Bns   client.BnsClient
	Confs map[string]func() gconf.Configuration
}

func (h *GconfHandler) knownConfigurations() []string {
	known := make([]string, 0, len(h.Confs))
	for name := range h.Confs {
		known = append(known, name)
	}
	sort.Strings(known)
	return known
}

func (h *GconfHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	extensionName := lastChunk(r.URL.Path)
	if extensionName == "" {
		JSONErr(w, http.StatusNotFound,
			fmt.Sprintf("Extension name must be provided. Supported extensions are %q", h.knownConfigurations()))
		return
	}

	var conf gconf.Configuration
	if fn, ok := h.Confs[extensionName]; ok {
		conf = fn()
	} else {
		log.Printf("extension %q gconf configuration entity unknown to gconf handler", extensionName)
		JSONErr(w, http.StatusNotFound,
			fmt.Sprintf("Configuration not registered for browsing. Supported extensions are %q", h.knownConfigurations()))
		return
	}

	switch err := client.ABCIKeyQuery(r.Context(), h.Bns, "/gconf", []byte(extensionName), conf); {
	case err == nil:
		JSONResp(w, http.StatusOK, conf)
	case errors.ErrNotFound.Is(err):
		JSONErr(w, http.StatusNotFound, http.StatusText(http.StatusNotFound))
	default:
		log.Printf("gconf ABCI query: %s", err)
		JSONErr(w, http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))
	}
}

type InfoHandler struct{}

// InfoHandler godoc
// @Summary Returns information about this instance of `bnsapi`.
// @Success 200
// @Router /info/ [get]
func (h *InfoHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	JSONResp(w, http.StatusOK, struct {
		BuildHash    string `json:"build_hash"`
		BuildVersion string `json:"build_version"`
	}{
		BuildHash:    util.BuildHash,
		BuildVersion: util.BuildVersion,
	})
}

type BlocksHandler struct {
	Bns client.BnsClient
}

// BlocksHandler godoc
// @Summary Get block details by height
// @Description get block detail by blockHeight
// @Param blockHeight path int true "Block Height"
// @Success 200 {object} json.RawMessage
// @Failure 404 {object} json.RawMessage
// @Redirect 303
// @Router /blocks/{blockHeight} [get]
func (h *BlocksHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	heightStr := lastChunk(r.URL.Path)
	if heightStr == "" {
		JSONRedirect(w, http.StatusSeeOther, "/blocks/1")
		return
	}
	height, err := strconv.ParseInt(heightStr, 10, 64)
	if err != nil {
		JSONErr(w, http.StatusNotFound, "block height must be a number")
		return
	}

	// We do not care about payload, proxy all!
	var payload json.RawMessage
	if err := h.Bns.Get(r.Context(), fmt.Sprintf("/block?height=%d", height), &payload); err != nil {
		log.Printf("Bns block height info: %s", err)
		JSONErr(w, http.StatusBadGateway, http.StatusText(http.StatusBadGateway))
		return
	}
	JSONResp(w, http.StatusOK, payload)
}

// lastChunk returns last path chunk - everything after the last `/` character.
// For example LAST in /foo/bar/LAST and empty string in /foo/bar/
func lastChunk(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' {
			return path[i+1:]
		}
	}
	return path
}

// DefaultHandler is used to handle the request that no other handler wants.
type DefaultHandler struct{}

func (h *DefaultHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// No trailing slash.
	if len(r.URL.Path) > 1 && r.URL.Path[len(r.URL.Path)-1] == '/' {
		path := strings.TrimRight(r.URL.Path, "/")
		JSONRedirect(w, http.StatusPermanentRedirect, path)
		return
	}
	JSONErr(w, http.StatusNotFound, http.StatusText(http.StatusNotFound))
}

type AccountDomainsHandler struct {
	Bns client.BnsClient
}

// AccountDomainsHandler godoc
// @Summary Returns a list of `bnsd/x/account` Domain entities.
// @Param admin query string false "Address of the admin"
// @Param offset query string false "Pagination offset"
// @Success 200 {object} json.RawMessage
// @Failure 404 {object} json.RawMessage
// @Redirect 303
// @Router /accounts/domains/ [get]
func (h *AccountDomainsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var it client.ABCIIterator
	q := r.URL.Query()
	offset := extractIDFromKey(q.Get("offset"))
	if admin := q.Get("admin"); len(admin) > 0 {
		rawAddr, err := weave.ParseAddress(admin)
		if err != nil {
			JSONErr(w, http.StatusBadRequest, "Admin address must be a valid address value..")
			return
		}
		end := nextKeyValue(rawAddr)
		it = client.ABCIRangeQuery(r.Context(), h.Bns, "/domains/admin", fmt.Sprintf("%s:%x:%x", admin, offset, end))
	} else {
		it = client.ABCIRangeQuery(r.Context(), h.Bns, "/domains", fmt.Sprintf("%x:", offset))
	}

	objects := make([]KeyValue, 0, paginationMaxItems)
fetchDomains:
	for {
		var model account.Domain
		switch key, err := it.Next(&model); {
		case err == nil:
			objects = append(objects, KeyValue{
				Key:   key,
				Value: &model,
			})
			if len(objects) == paginationMaxItems {
				break fetchDomains
			}
		case errors.ErrIteratorDone.Is(err):
			break fetchDomains
		default:
			log.Printf("account domain ABCI query: %s", err)
			JSONErr(w, http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))
			return
		}
	}
	JSONResp(w, http.StatusOK, struct {
		Objects []KeyValue `json:"objects"`
	}{
		Objects: objects,
	})
}

type AccountsAccountsDetailHandler struct {
	Bns client.BnsClient
}

// AccountsAccountsDetailHandler godoc
// @Summary Returns a list of `bnsd/x/account` Account entitiy.
// @Param accountKey path string false "Address of the admin"
// @Success 200 {object} json.RawMessage
// @Failure 404 {object} json.RawMessage
// @Failure 500 {object} json.RawMessage
// @Router /accounts/accounts/{accountKey} [get]
func (h *AccountsAccountsDetailHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	accountKey := lastChunk(r.URL.Path)
	var acc account.Account
	switch err := client.ABCIKeyQuery(r.Context(), h.Bns, "/accounts", []byte(accountKey), &acc); {
	case err == nil:
		JSONResp(w, http.StatusOK, acc)
	case errors.ErrNotFound.Is(err):
		JSONErr(w, http.StatusNotFound, http.StatusText(http.StatusNotFound))
	default:
		log.Printf("account ABCI query: %s", err)
		JSONErr(w, http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))
	}
}

// AccountAccountsDetailHandler godoc
// @Summary Returns a list of `bnsd/x/account` Account entitiy.
// @Param accountKey path string false "Address of the admin"
// @Success 200 {object} json.RawMessage
// @Failure 404 {object} json.RawMessage
// @Failure 500 {object} json.RawMessage
// @Router /accounts/accounts/{accountKey} [get]
type AccountsAccountsHandler struct {
	Bns client.BnsClient
}

func (h *AccountsAccountsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	if !atMostOne(q, "domain", "owner") {
		JSONErr(w, http.StatusBadRequest, "At most one filter can be used at a time.")
		return
	}

	var it client.ABCIIterator
	offset := extractIDFromKey(q.Get("offset"))
	if d := q.Get("domain"); len(d) > 0 {
		end := nextKeyValue([]byte(d))
		it = client.ABCIRangeQuery(r.Context(), h.Bns, "/accounts/domain", fmt.Sprintf("%x:%x:%x", d, offset, end))
	} else if o := q.Get("owner"); len(o) > 0 {
		rawAddr, err := weave.ParseAddress(o)
		if err != nil {
			JSONErr(w, http.StatusBadRequest, "Owner address must be a valid address value..")
			return
		}
		end := nextKeyValue(rawAddr)
		it = client.ABCIRangeQuery(r.Context(), h.Bns, "/accounts/owner", fmt.Sprintf("%s:%x:%x", o, offset, end))
	} else {
		it = client.ABCIRangeQuery(r.Context(), h.Bns, "/accounts", fmt.Sprintf("%x:", offset))
	}

	objects := make([]KeyValue, 0, paginationMaxItems)
fetchAccounts:
	for {
		var acc account.Account
		switch key, err := it.Next(&acc); {
		case err == nil:
			objects = append(objects, KeyValue{
				Key:   key,
				Value: &acc,
			})
			if len(objects) == paginationMaxItems {
				break fetchAccounts
			}
		case errors.ErrIteratorDone.Is(err):
			break fetchAccounts
		default:
			log.Printf("account account ABCI query: %s", err)
			JSONErr(w, http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))
			return
		}
	}

	JSONResp(w, http.StatusOK, struct {
		Objects []KeyValue `json:"objects"`
	}{
		Objects: objects,
	})
}

// atMostOne returns true if at most one non empty value from given list of
// names exists in the query.
func atMostOne(query url.Values, names ...string) bool {
	var nonempty int
	for _, name := range names {
		if query.Get(name) != "" {
			nonempty++
		}
		if nonempty > 1 {
			return false
		}
	}
	return true
}

func extractIDFromKey(key string) []byte {
	raw, err := weave.ParseAddress(key)
	if err != nil {
		// Cannot decode, return everything.
		return []byte(key)
	}
	for i, c := range raw {
		if c == ':' {
			return raw[i+1:]
		}
	}
	return raw
}

// paginationMaxItems defines how many items should a single result return.
// This values should not be greater than orm.queryRangeLimit so that each
// query returns enough results.
const paginationMaxItems = 50

type KeyValue struct {
	Key   hexbytes  `json:"key"`
	Value orm.Model `json:"value"`
}

// hexbytes is a byte type that JSON serialize to hex encoded string.
type hexbytes []byte

func (b hexbytes) MarshalJSON() ([]byte, error) {
	return json.Marshal(hex.EncodeToString(b))
}

func (b *hexbytes) UnmarshalJSON(enc []byte) error {
	var s string
	if err := json.Unmarshal(enc, &s); err != nil {
		return err
	}
	val, err := hex.DecodeString(s)
	if err != nil {
		return err
	}
	*b = val
	return nil
}

// JSONResp write content as JSON encoded response.
func JSONResp(w http.ResponseWriter, code int, content interface{}) {
	b, err := json.MarshalIndent(content, "", "\t")
	if err != nil {
		log.Printf("cannot JSON serialize response: %s", err)
		code = http.StatusInternalServerError
		b = []byte(`{"errors":["Internal Server Errror"]}`)
	}
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(code)

	const MB = 1 << (10 * 2)
	if len(b) > MB {
		log.Printf("response JSON body is huge: %d", len(b))
	}
	_, _ = w.Write(b)
}

// JSONErr write single error as JSON encoded response.
func JSONErr(w http.ResponseWriter, code int, errText string) {
	JSONErrs(w, code, []string{errText})
}

// JSONErrs write multiple errors as JSON encoded response.
func JSONErrs(w http.ResponseWriter, code int, errs []string) {
	resp := struct {
		Errors []string `json:"errors"`
	}{
		Errors: errs,
	}
	JSONResp(w, code, resp)
}

// JSONRedirect return redirect response, but with JSON formatted body.
func JSONRedirect(w http.ResponseWriter, code int, urlStr string) {
	w.Header().Set("Location", urlStr)
	var content = struct {
		Code     int
		Location string
	}{
		Code:     code,
		Location: urlStr,
	}
	JSONResp(w, code, content)
}

func nextKeyValue(b []byte) []byte {
	if len(b) == 0 {
		return nil
	}
	next := make([]byte, len(b))
	copy(next, b)

	// If the last value does not overflow, increment it. Otherwise this is
	// an edge case and key must be extended.
	if next[len(next)-1] < math.MaxUint8 {
		next[len(next)-1]++
	} else {
		next = append(next, 0)
	}
	return next
}

func encodeSequence(val uint64) []byte {
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, val)
	return bz
}
