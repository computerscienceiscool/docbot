package google

import (
	"encoding/json"
	"io/ioutil"
	"strings"
	"testing"

	// "github.com/sergi/go-diff/diffmatchpatch"

	. "github.com/stevegt/goadapt"
)

// regenerate testdata
const regen bool = true

const (
	credpath = "../../local/mcpbot-mcpbot-key.json"
	folderId = "1HcCIw7ppJZPD9GEHccnkgNYUwhAGCif6"
)

func setup(t *testing.T) (f *Folder) {
	cbuf, err := ioutil.ReadFile(credpath)
	Tassert(t, err == nil, err)
	f, err = NewFolder(cbuf, folderId, "mcp")
	Tassert(t, err == nil, err)
	return
}

func TestContent(t *testing.T) {
	gf := setup(t)

	fn := "mcp-4-why-numbered-docs"
	expect := "Name: mcp-4-why-numbered-docs\n"

	node, err := gf.Getnode(fn)
	Tassert(t, err == nil, err)
	Tassert(t, node != nil, Spf("%#v", node))

	// https://github.com/rsbh/doc2md/blob/a740060638ca55813c25c7e4a6cf7774e3cbd63f/pkg/transformer/doc2json.go#L368
	doc, err := gf.docs.Documents.Get(node.Id()).Do()
	Tassert(t, err == nil, err)
	b := doc.Body
	// iterate over elements
	var got string
	for _, s := range b.Content {
		if s.Paragraph != nil {
			for _, el := range s.Paragraph.Elements {
				if el.TextRun != nil {
					content := el.TextRun.Content
					// Pprint(content)
					if got == "" && strings.HasPrefix(content, "Name: ") {
						got = content
					}
				}
			}
		}
	}
	Tassert(t, got == expect, got)

	// save a copy of content for reverse engineering
	buf, err := json.MarshalIndent(b.Content, "", "  ")
	Tassert(t, err == nil, err)
	err = ioutil.WriteFile("/tmp/mcp-4.json", buf, 0644)
	Tassert(t, err == nil, err)

}
