package shared

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/lroolle/orgx-cli/pkg/ir"
	"github.com/lroolle/orgx-cli/pkg/parser"
)

type RefType string

const (
	RefTypeID      RefType = "id"
	RefTypeOutline RefType = "outline"
	RefTypeHash    RefType = "hash"
	RefTypePath    RefType = "path"
)

type Ref struct {
	Path    string
	RefType RefType
	Value   string
}

var (
	idRefPattern      = regexp.MustCompile(`^(.+)::ID:(.+)$`)
	outlineRefPattern = regexp.MustCompile(`^(.+)::/(.+)$`)
	hashRefPattern    = regexp.MustCompile(`^(.+)::H:([a-f0-9]+)$`)
)

func ParseRefFromArg(arg string) (Ref, error) {
	if m := idRefPattern.FindStringSubmatch(arg); m != nil {
		return Ref{Path: m[1], RefType: RefTypeID, Value: m[2]}, nil
	}

	if m := hashRefPattern.FindStringSubmatch(arg); m != nil {
		return Ref{Path: m[1], RefType: RefTypeHash, Value: m[2]}, nil
	}

	if m := outlineRefPattern.FindStringSubmatch(arg); m != nil {
		return Ref{Path: m[1], RefType: RefTypeOutline, Value: m[2]}, nil
	}

	return Ref{Path: arg, RefType: RefTypePath}, nil
}

func (r Ref) String() string {
	switch r.RefType {
	case RefTypeID:
		return fmt.Sprintf("%s::ID:%s", r.Path, r.Value)
	case RefTypeHash:
		return fmt.Sprintf("%s::H:%s", r.Path, r.Value)
	case RefTypeOutline:
		return fmt.Sprintf("%s::/%s", r.Path, r.Value)
	default:
		return r.Path
	}
}

func FindHeading(ref Ref) (*ir.Heading, error) {
	doc, err := parser.ParseFile(ref.Path)
	if err != nil {
		return nil, err
	}

	return findHeadingInNodes(doc.Nodes, ref)
}

func findHeadingInNodes(nodes []ir.Node, ref Ref) (*ir.Heading, error) {
	for _, n := range nodes {
		if h, ok := n.(*ir.Heading); ok {
			if matchesRef(h, ref) {
				return h, nil
			}
			if len(h.Children) > 0 {
				if found, err := findHeadingInNodes(h.Children, ref); err == nil {
					return found, nil
				}
			}
		}
	}
	return nil, fmt.Errorf("heading not found: %s", ref.String())
}

func matchesRef(h *ir.Heading, ref Ref) bool {
	switch ref.RefType {
	case RefTypeID:
		if id, ok := h.Props["ID"]; ok {
			return id == ref.Value
		}
		return false
	case RefTypeHash:
		return strings.Contains(h.Ref, "::H:"+ref.Value)
	case RefTypeOutline:
		return strings.Contains(h.Ref, "::/"+ref.Value)
	default:
		return false
	}
}

var HeadingFields = []string{
	"ref",
	"level",
	"title",
	"todo",
	"tags",
	"props",
	"scheduled",
	"deadline",
	"body",
	"span",
}
