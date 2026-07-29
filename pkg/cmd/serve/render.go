package serve

import (
	"bytes"
	"html/template"
	"io"
	"log"
	"os"

	"github.com/niklasfasching/go-org/org"
)

// renderOrg renders one org file to HTML. idBase becomes the URL
// prefix for [[id:...]] links (go-org's link-protocol table does
// the rewriting), so graph links navigate the preview instead of
// dead-ending on an id: URL. Rendering never touches the file —
// reads never write, and nothing rendered is cached.
func renderOrg(path, idBase string) (template.HTML, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	conf := org.New()
	// The preview shows the file, not a document export: no TOC.
	conf.DefaultSettings["OPTIONS"] = "toc:nil <:t e:t f:t pri:t todo:t tags:t title:t ealb:nil"
	conf.Log = log.New(io.Discard, "", 0)
	doc := conf.Parse(bytes.NewReader(raw), path)
	doc.Links["id"] = idBase + "%s"
	out, err := doc.Write(org.NewHTMLWriter())
	if err != nil {
		return "", err
	}
	return template.HTML(out), nil
}
