package cmdutil

import "github.com/lroolle/orgx-cli/pkg/iostreams"

type Factory struct {
	AppVersion string
	IOStreams  *iostreams.IOStreams
	Prompter   Prompter
}

type Prompter interface {
	Confirm(prompt string) error
	Input(prompt string, defaultValue string) (string, error)
	Select(prompt string, options []string) (int, error)
}
