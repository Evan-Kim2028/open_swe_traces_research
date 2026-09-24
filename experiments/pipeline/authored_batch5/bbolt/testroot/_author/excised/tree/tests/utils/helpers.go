package utils

import (
	"flag"
	_ "fmt"
	_ "os"
)

var enableRoot bool

func init() {
	flag.BoolVar(&enableRoot, "test.root", false, "enable tests that require root")
}

// RequiresRoot requires root and the test.root flag has been set.
func RequiresRoot() {
	panic("excised: RequiresRoot")
}
