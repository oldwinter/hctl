package main

import (
	"fmt"
	"os"

	"github.com/oldwinter/harnessctl/internal/cli"
	"github.com/oldwinter/harnessctl/internal/exitcode"
)

func main() {
	if err := cli.NewRoot().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(exitcode.From(err))
	}
}
