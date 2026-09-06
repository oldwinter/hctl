package main

import (
	"fmt"
	"io"
	"os"

	"github.com/oldwinter/hctl/internal/cli"
	"github.com/oldwinter/hctl/internal/exitcode"
)

// osExit is stubbed in tests so main can be covered without killing the test process.
var osExit = os.Exit

func main() {
	osExit(run(os.Args[1:], os.Stderr))
}

func run(args []string, errW io.Writer) int {
	cmd := cli.NewRoot()
	cmd.SetArgs(args)
	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(errW, err)
		return exitcode.From(err)
	}
	return 0
}
