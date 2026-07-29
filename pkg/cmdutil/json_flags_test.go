package cmdutil

import (
	"encoding/json"
	"testing"

	"github.com/spf13/cobra"
)

func TestEnvelopeInjectsKindIntoObjects(t *testing.T) {
	out := envelope("orgx.node.list.v1", map[string]any{"count": 2})
	raw, _ := json.Marshal(out)
	var got map[string]any
	_ = json.Unmarshal(raw, &got)
	if got["kind"] != "orgx.node.list.v1" || got["count"] != float64(2) {
		t.Fatalf("envelope = %s", raw)
	}
}

func TestEnvelopeWrapsArraysWithCount(t *testing.T) {
	out := envelope("orgx.find.v1", []string{"a", "b"})
	raw, _ := json.Marshal(out)
	var got struct {
		Kind  string   `json:"kind"`
		Count int      `json:"count"`
		Items []string `json:"items"`
	}
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	if got.Kind != "orgx.find.v1" || got.Count != 2 || len(got.Items) != 2 {
		t.Fatalf("envelope = %s", raw)
	}
}

func TestEnvelopeTreatsNilAsEmptyResult(t *testing.T) {
	var none []string
	raw, _ := json.Marshal(envelope("orgx.find.v1", none))
	var got struct {
		Count int   `json:"count"`
		Items []any `json:"items"`
	}
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	if got.Count != 0 || got.Items == nil {
		t.Fatalf("nil result must be an empty envelope, got %s", raw)
	}
}

func TestKindFollowsCommandPath(t *testing.T) {
	root := &cobra.Command{Use: "orgx"}
	node := &cobra.Command{Use: "node"}
	list := &cobra.Command{Use: "list"}
	node.AddCommand(list)
	root.AddCommand(node)
	if got := KindFor(list); got != "orgx.node.list.v1" {
		t.Fatalf("kind = %q", got)
	}
}
