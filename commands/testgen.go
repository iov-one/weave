package commands

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/gogo/protobuf/proto"
)

// Example will be written out to a file, .json and .bin
// Filename should have no path and no extension
type Example struct {
	Filename string
	Obj      proto.Message
}

// TestGenCmd generates sample protobuf and json encodings
// of various objects to test against.
func TestGenCmd(examples []Example, args []string) error {
	outdir := "testdata"
	if len(args) > 0 {
		outdir = args[0]
	}
	err := os.MkdirAll(outdir, 0755)
	if err != nil {
		return err
	}

	for _, ex := range examples {
		// write json data
		js, err := json.Marshal(ex.Obj)
		if err != nil {
			return err
		}
		jsFile := filepath.Join(outdir, ex.Filename+".json")
		err = ioutil.WriteFile(jsFile, js, 0644)
		if err != nil {
			return err
		}

		// write protbuf data
		pb, err := proto.Marshal(ex.Obj)
		if err != nil {
			return err
		}
		pbFile := filepath.Join(outdir, ex.Filename+".bin")
		err = ioutil.WriteFile(pbFile, pb, 0644)
		if err != nil {
			return err
		}
	}
	return nil
}
