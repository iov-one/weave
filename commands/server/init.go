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

	"github.com/pkg/errors"
	"github.com/tendermint/tmlibs/log"
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

/*
Usage:
  xxx init // index all
  xxx init -all=f  // no index
  xxx init -tags=foo,bar // index only foo and bar
*/
func parseIndex(args []string) (bool, bool, string, bool, []string, error) {
	// parse flagIndexAll, flagIndexTags and return the result
	indexFlags := flag.NewFlagSet("init", flag.ExitOnError)
	tags := indexFlags.String(flagIndexTags, "", "comma-separated list of tags to index")
	all := indexFlags.Bool(flagIndexAll, true, "")
	force := indexFlags.Bool(FlagForce, false, "")
	ignore := indexFlags.Bool(FlagIgnore, false, "")

	err := indexFlags.Parse(args)
	return *all, *force, *tags, *ignore, indexFlags.Args(), err
}

// InitCmd will initialize all files for tendermint,
// along with proper app_options.
// The application can pass in a function to generate
// proper options. And may want to use GenerateCoinKey
// to create default account(s).
func InitCmd(gen GenOptions, logger log.Logger, home string, args []string) error {
	genFile := filepath.Join(home, DirConfig, "genesis.json")
	confFile := filepath.Join(home, DirConfig, "config.toml")

	all, force, tags, ignore, args, err := parseIndex(args)
	if err != nil {
		return err
	}
	err = setTxIndex(confFile, all, tags, force, ignore)
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
	err = addGenesisOptions(genFile, options, force, ignore)
	if err == nil {
		fmt.Println("The application has been succesfully initialised.")
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
		return fmt.Errorf(ErrorAlreadyInitialised, FlagForce, FlagIgnore)
	}

	timeJson, _ := time.Now().MarshalJSON()

	doc[AppStateKey] = options
	doc[GenesisTimeKey] = timeJson

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
func setTxIndex(config string, all bool, tags string, force, ignore bool) error {
	f, err := os.Open(config)
	if err != nil {
		return errors.WithStack(err)
	}

	// translate the file into a buffer in memory
	scan := bufio.NewScanner(f)
	var buf []string
	for scan.Scan() {
		line := scan.Text()
		if strings.HasPrefix(line, prefixIndexer) {
			line = setIndexer
		} else if strings.HasPrefix(line, prefixIndexAll) {
			line = fmt.Sprintf("%s = %t", prefixIndexAll, all)
		} else if strings.HasPrefix(line, prefixIndexTags) {
			line = fmt.Sprintf(`%s = "%s"`, prefixIndexTags, tags)
		}
		buf = append(buf, line)
	}
	buf = append(buf, "")
	f.Close()

	// write to output
	out, err := os.Create(config)
	if err != nil {
		return errors.WithStack(err)
	}
	output := strings.Join(buf, "\n")
	_, err = out.WriteString(output)
	out.Close()
	return err
}
