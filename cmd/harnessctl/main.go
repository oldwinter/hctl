package main

import (
	"fmt"
	"os"

	"github.com/oldwinter/hctl/internal/cli"
	"github.com/oldwinter/hctl/internal/exitcode"
)

func main() {
	if err := cli.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(exitcode.From(err))
	}
}
