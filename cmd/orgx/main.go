package main

import (
	"encoding/json"
	"fmt"
	"os"
	"slices"

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
		if err == cmdutil.SilentError || err == cmdutil.CancelError {
			return 1
		}
		// In --json mode errors are an envelope too (orgx.error.v1,
		// on stderr — stdout stays data-only), so an agent gets a
		// machine-readable message and, when known, the fix.
		if slices.Contains(os.Args[1:], "--json") {
			enc := json.NewEncoder(f.IOStreams.ErrOut)
			enc.SetIndent("", "  ")
			_ = enc.Encode(cmdutil.NewErrorEnvelope(err))
			return 1
		}
		fmt.Fprintln(f.IOStreams.ErrOut, err)
		if cmdutil.IsFlagError(err) {
			cmd.Usage()
		}
		return 1
	}

	return 0
}
