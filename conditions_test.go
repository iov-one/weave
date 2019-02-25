package weave_test

import (
	"fmt"
	"testing"

	"github.com/iov-one/weave"
	. "github.com/smartystreets/goconvey/convey"
)

func TestAddressPrinting(t *testing.T) {
	Convey("test hexademical address printing", t, func() {
		b := []byte("ABCD123456LHB")
		addr := weave.Address(b)

		So(addr.String(), ShouldNotEqual, fmt.Sprintf("%X", addr))
	})

	Convey("test hexademical condition printing", t, func() {
		cond := weave.NewCondition("12", "32", []byte("ABCD123456LHB"))

		So(cond.String(), ShouldNotEqual, fmt.Sprintf("%X", cond))
	})
}
