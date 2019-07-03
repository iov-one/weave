package server

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/iov-one/weave/errors"
	"github.com/tendermint/tendermint/libs/log"
)

const (
	// AppStateKey is the key in the json json where all info
	// on initializing the app can be found
	AppStateKey             = "app_state"
	DirConfig               = "config"
	GenesisTimeKey          = "genesis_time"
	ErrorAlreadyInitialised = "the application has already been initialised, use %s flag to override or %s to ignore"
	FlagForce               = "f"
	FlagIgnore              = "i"
	flagIndexAll            = "all"
	flagIndexTags           = "tags"
)

type indexFlagValues struct {
	tags     string
	indexAll bool
	force    bool
	ignore   bool
}

/*
Usage:
  xxx init // index all
  xxx init -all=f  // no index
  xxx init -tags=foo,bar // index only foo and bar
*/
func parseIndex(args []string) (indexFlagValues, []string, error) {
	vals := indexFlagValues{}
	// parse flagIndexAll, flagIndexTags and return the result
	indexFlags := flag.NewFlagSet("init", flag.ExitOnError)
	indexFlags.StringVar(&vals.tags, flagIndexTags, "", "comma-separated list of tags to index")
	indexFlags.BoolVar(&vals.indexAll, flagIndexAll, true, "")
	indexFlags.BoolVar(&vals.force, FlagForce, false, "")
	indexFlags.BoolVar(&vals.ignore, FlagIgnore, false, "")

	err := indexFlags.Parse(args)
	return vals, indexFlags.Args(), err
}

// InitCmd will initialize all files for tendermint,
// along with proper app_options.
// The application can pass in a function to generate
// proper options. And may want to use GenerateCoinKey
// to create default account(s).
func InitCmd(gen GenOptions, logger log.Logger, home string, args []string) error {
	genFile := filepath.Join(home, DirConfig, "genesis.json")
	confFile := filepath.Join(home, DirConfig, "config.toml")

	vals, args, err := parseIndex(args)
	if err != nil {
		return err
	}
	err = setTxIndex(confFile, vals)
	if err != nil {
		return err
	}

	// no app_options, leave like tendermint
	if gen == nil {
		return nil
	}

	// Now, we want to add the custom app_options
	options, err := gen(args)
	if err != nil {
		return err
	}

	// And add them to the genesis file
	err = addGenesisOptions(genFile, options, vals.force, vals.ignore)
	if err == nil {
		fmt.Println("The application has been successfully initialised.")
	}

	return err
}

// GenOptions can parse command-line and flag to
// generate default app_options for the genesis file.
// This is application-specific
type GenOptions func(args []string) (json.RawMessage, error)

// GenesisDoc involves some tendermint-specific structures we don't
// want to parse, so we just grab it into a raw object format,
// so we can add one line.
type GenesisDoc map[string]json.RawMessage

func addGenesisOptions(filename string, options json.RawMessage, force, ignore bool) error {
	bz, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}

	var doc GenesisDoc
	err = json.Unmarshal(bz, &doc)
	if err != nil {
		return err
	}

	v, ok := doc[AppStateKey]
	if !force && ok && len(v) > 0 {
		if ignore {
			return nil
		}
		return errors.Wrapf(errors.ErrState, ErrorAlreadyInitialised, FlagForce, FlagIgnore)
	}

	timeJSON, _ := time.Now().UTC().MarshalJSON()

	doc[AppStateKey] = options
	doc[GenesisTimeKey] = timeJSON

	out, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}

	return ioutil.WriteFile(filename, out, 0600)
}

var (
	prefixIndexer   = "indexer"
	prefixIndexAll  = "index_all_tags"
	prefixIndexTags = "index_tags"

	setIndexer = `indexer = "kv"`
)

// setTxIndex sets the following fields in config.toml
//   indexer = "kv"
//   index_all_tags = <all>
//   index_tags = <tags>
func setTxIndex(config string, vals indexFlagValues) error {
	f, err := os.Open(config)
	if err != nil {
		return errors.Wrap(err, "unable to open file")
	}

	// translate the file into a buffer in memory
	scan := bufio.NewScanner(f)
	var buf []string
	for scan.Scan() {
		line := scan.Text()
		if strings.HasPrefix(line, prefixIndexer) {
			line = setIndexer
		} else if strings.HasPrefix(line, prefixIndexAll) {
			line = fmt.Sprintf("%s = %t", prefixIndexAll, vals.indexAll)
		} else if strings.HasPrefix(line, prefixIndexTags) {
			line = fmt.Sprintf(`%s = "%s"`, prefixIndexTags, vals.tags)
		}
		buf = append(buf, line)
	}
	buf = append(buf, "")
	f.Close()

	// write to output
	out, err := os.Create(config)
	if err != nil {
		return errors.Wrap(err, "unable to create file")
	}
	output := strings.Join(buf, "\n")
	_, err = out.WriteString(output)
	out.Close()
	return err
}
