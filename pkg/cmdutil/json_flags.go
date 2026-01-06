package cmdutil

import (
	"encoding/json"

	"github.com/lroolle/orgx-cli/pkg/iostreams"
	"github.com/spf13/cobra"
)

type Exporter interface {
	Write(io *iostreams.IOStreams, data interface{}) error
	Fields() []string
}

type jsonExporter struct {
	fields []string
}

func (e *jsonExporter) Write(io *iostreams.IOStreams, data interface{}) error {
	encoder := json.NewEncoder(io.Out)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}

func (e *jsonExporter) Fields() []string {
	return e.fields
}

func AddJSONFlags(cmd *cobra.Command, target *Exporter, defaultFields []string) {
	var jsonFlag bool

	cmd.Flags().BoolVar(&jsonFlag, "json", false, "Output as JSON")

	cmd.PreRunE = func(cmd *cobra.Command, args []string) error {
		if jsonFlag {
			*target = &jsonExporter{fields: defaultFields}
		}
		return nil
	}
}
