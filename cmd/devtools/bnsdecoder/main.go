package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"

	"github.com/iov-one/weave/cmd/bnsd/app"
)

func main() {
	if len(os.Args) != 2 || len(os.Args[1]) == 0 {
		_, _ = fmt.Fprint(os.Stderr, "bawe 64vencoded tx argument required")
		os.Exit(1)
	}
	tx, err := base64.StdEncoding.DecodeString(os.Args[1])
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Failed to base64 decode tx: %s\n", err)
		os.Exit(1)
	}
	var protoTx app.Tx
	if err := protoTx.Unmarshal(tx); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Failed to unmarshall protobuf tx: %s\n", err)
		os.Exit(1)
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent(" ", " ")
	if err := encoder.Encode(&protoTx); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Failed to print json for protobuf tx: %s\n", err)
		os.Exit(1)
	}
}
