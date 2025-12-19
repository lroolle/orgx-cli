package parser

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/lroolle/orgx-cli/pkg/ir"
)

type Parser interface {
	Parse(path string, content []byte) (*ir.Document, error)
	CanParse(path string) bool
}

func ParseFile(path string) (*ir.Document, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	ext := strings.ToLower(filepath.Ext(path))
	var parser Parser

	switch ext {
	case ".org":
		parser = &OrgParser{}
	case ".md", ".markdown":
		parser = &MarkdownParser{}
	default:
		return nil, fmt.Errorf("unsupported file type: %s", ext)
	}

	return parser.Parse(path, content)
}

func computeSHA256(content []byte) string {
	h := sha256.Sum256(content)
	return fmt.Sprintf("%x", h)
}
