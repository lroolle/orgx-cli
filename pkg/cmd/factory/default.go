package factory

import (
	"github.com/lroolle/orgx-cli/pkg/cmdutil"
	"github.com/lroolle/orgx-cli/pkg/iostreams"
)

func New(version string) *cmdutil.Factory {
	f := &cmdutil.Factory{
		AppVersion: version,
	}

	f.IOStreams = iostreams.System()

	return f
}
