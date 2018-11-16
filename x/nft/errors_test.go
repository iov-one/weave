package nft

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPrintableId(t *testing.T) {
	cases := []struct {
		id       []byte
		expected string
	}{
		// print as strings
		{[]byte("ATOM"), "ATOM"},
		{[]byte("test-chain-Ad6d2dD"), "test-chain-Ad6d2dD"},
		// print as hex (special chars)
		{[]byte{0x88, 0x99, 0xad, 0x00}, "0x8899ad00"},
		// newline, or any control chars should also trigger hex
		{[]byte{0x43, 0x55, 0x0a, 0x57}, "0x43550a57"},
		{nil, "<nil>"},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("case %d", i), func(tt *testing.T) {
			printable := printableId(tc.id)
			assert.Equal(t, tc.expected, printable)
		})
	}
}
