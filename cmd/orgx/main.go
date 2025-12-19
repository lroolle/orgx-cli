package main

import (
	"fmt"
	"os"

	"github.com/lroolle/orgx-cli/pkg/cmd/factory"
	"github.com/lroolle/orgx-cli/pkg/cmd/root"
	"github.com/lroolle/orgx-cli/pkg/cmdutil"
)

var version = "0.1.0-dev"

func main() {
	code := run()
	os.Exit(code)
}

func run() int {
	f := factory.New(version)
	cmd := root.NewCmdRoot(f)

	if err := cmd.Execute(); err != nil {
		if cmdutil.IsFlagError(err) {
			fmt.Fprintln(f.IOStreams.ErrOut, err)
			cmd.Usage()
			return 1
		}
		if err != cmdutil.SilentError && err != cmdutil.CancelError {
			fmt.Fprintln(f.IOStreams.ErrOut, err)
		}
		return 1
	}

	return 0
}
