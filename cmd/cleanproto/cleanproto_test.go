package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestFormat(t *testing.T) {
	const proto = `
// This is an example protobuf file.
package foobar;

import "github.com/iov-one/weave/x/paychan/codec.proto";
import "github.com/gogo/protobuf/gogoproto/gogo.proto";
import "github.com/iov-one/weave/cmd/bnsd/x/nft/username/codec.proto";

option go_package = "foobar";

// SendMsg is a request to move these coins from the given
// source to the given destination address.
// memo is an optional human-readable message
// ref is optional binary data, that can refer to another
// eg. tx hash
message SendMsg {
  option (gogoproto.goproto_stringer) = false;
  weave.Metadata metadata = 1;
  bytes src = 2 [
  	(gogoproto.casttype) = "github.com/iov-one/weave.Address"
  	(gogoproto.customname) = "Source"
  ];
  bytes dest = 3 [
     (gogoproto.casttype)

     =

     "github.com/iov-one/weave.Address"];
  coin.Coin amount = 4;
  // max length 128 character
  string memo = 5;
  // max length 64 bytes
  bytes ref = 6;
}
	`
	var out bytes.Buffer
	err := cleanup(strings.NewReader(proto), &out)
	if err != nil {
		t.Fatalf("cleanup failed: %s", err)
	}

	const wantDecl = `
// This is an example protobuf file.
package foobar;

import "github.com/iov-one/weave/x/paychan/codec.proto";
import "github.com/iov-one/weave/cmd/bnsd/x/nft/username/codec.proto";

// SendMsg is a request to move these coins from the given
// source to the given destination address.
// memo is an optional human-readable message
// ref is optional binary data, that can refer to another
// eg. tx hash
message SendMsg {
  weave.Metadata metadata = 1;
  bytes src = 2 ;
  bytes dest = 3 ;
  coin.Coin amount = 4;
  // max length 128 character
  string memo = 5;
  // max length 64 bytes
  bytes ref = 6;
}
	`
	if gotDecl := out.String(); gotDecl != wantDecl {
		t.Logf("want: \n%s", wantDecl)
		t.Logf("got: \n%s", gotDecl)
		t.Errorf("unexpected declaration resultresult")
	}
}
