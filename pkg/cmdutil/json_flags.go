package cmdutil

import (
	"encoding/json"
	"strings"

	"github.com/lroolle/orgx-cli/pkg/iostreams"
	"github.com/spf13/cobra"
)

type Exporter interface {
	Write(io *iostreams.IOStreams, data interface{}) error
	Fields() []string
}

// jsonExporter writes the versioned envelope: every object gains a
// "kind" (orgx.<command path>.v1); a bare array becomes
// {kind, count, items}. The kind names the schema so agents can
// dispatch on it and changes must be additive to stay v1.
type jsonExporter struct {
	kind   string
	fields []string
}

func (e *jsonExporter) Write(io *iostreams.IOStreams, data interface{}) error {
	encoder := json.NewEncoder(io.Out)
	encoder.SetIndent("", "  ")
	return encoder.Encode(envelope(e.kind, data))
}

func envelope(kind string, data interface{}) interface{} {
	raw, err := json.Marshal(data)
	if err != nil {
		return data // let the encoder surface the real error
	}
	switch {
	case len(raw) > 0 && raw[0] == '{':
		var obj map[string]interface{}
		if json.Unmarshal(raw, &obj) == nil {
			obj["kind"] = kind
			return obj
		}
	case len(raw) > 0 && raw[0] == '[':
		var items []interface{}
		if json.Unmarshal(raw, &items) == nil {
			return map[string]interface{}{"kind": kind, "count": len(items), "items": items}
		}
	case string(raw) == "null":
		// A nil slice is an empty result, not a null result.
		return map[string]interface{}{"kind": kind, "count": 0, "items": []interface{}{}}
	}
	return data
}

func (e *jsonExporter) Fields() []string {
	return e.fields
}

// KindFor derives the envelope kind from a command's path:
// "orgx node list" -> "orgx.node.list.v1". The orgx prefix is
// normalized so a command exercised standalone (tests, embedding)
// emits the same kind it does under the real root.
func KindFor(cmd *cobra.Command) string {
	kind := strings.ReplaceAll(cmd.CommandPath(), " ", ".")
	if !strings.HasPrefix(kind, "orgx.") {
		kind = "orgx." + kind
	}
	return kind + ".v1"
}

func AddJSONFlags(cmd *cobra.Command, target *Exporter, defaultFields []string) {
	var jsonFlag bool

	cmd.Flags().BoolVar(&jsonFlag, "json", false, "Output as JSON")

	cmd.PreRunE = func(cmd *cobra.Command, args []string) error {
		if jsonFlag {
			*target = &jsonExporter{kind: KindFor(cmd), fields: defaultFields}
		}
		return nil
	}
}
