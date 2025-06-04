package fastboot

import (
	"flag"
	"os"
	"testing"
)

var noSandboxFlag = flag.Bool("test.noSandbox", false, "disable sandbox mode for testing")

func TestMain(m *testing.M) {
	flag.Parse()
	noSandbox = *noSandboxFlag
	os.Exit(m.Run())
}
