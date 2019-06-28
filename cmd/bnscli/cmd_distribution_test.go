package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/iov-one/weave/weavetest/assert"
	"github.com/iov-one/weave/x/distribution"
)

func TestCmdResetRevenue(t *testing.T) {
	destinationsPath := mustCreateFile(t, strings.NewReader(`seq:foo/bar/1,3
seq:foo/bar/2,1
seq:foo/bar/3,20`))

	var output bytes.Buffer
	args := []string{
		"-revenue", "b1ca7e78f74423ae01da3b51e676934d9105f282",
		"-destinations", destinationsPath,
	}
	if err := cmdResetRevenue(nil, &output, args); err != nil {
		t.Fatalf("cannot create a transaction: %s", err)
	}

	tx, _, err := readTx(&output)
	if err != nil {
		t.Fatalf("cannot read created transaction: %s", err)
	}

	txmsg, err := tx.GetMsg()
	if err != nil {
		t.Fatalf("cannot get transaction message: %s", err)
	}
	msg := txmsg.(*distribution.ResetMsg)

	assert.Equal(t, msg.RevenueID, fromHex(t, "b1ca7e78f74423ae01da3b51e676934d9105f282"))
	assert.Equal(t, len(msg.Destinations), 3)
	assert.Equal(t, msg.Destinations[0].Weight, int32(3))
	assert.Equal(t, msg.Destinations[1].Weight, int32(1))
	assert.Equal(t, msg.Destinations[2].Weight, int32(20))
}
