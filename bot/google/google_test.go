package google

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	// "github.com/sergi/go-diff/diffmatchpatch"

	. "github.com/stevegt/goadapt"
)

// regenerate testdata
const regen bool = false

const (
	credpath = "../../local/mcpbot-mcpbot-key.json"
	folderId = "1HcCIw7ppJZPD9GEHccnkgNYUwhAGCif6"
)

func TestMain(m *testing.M) {
	setup()
	code := m.Run()
	// teardown()
	os.Exit(code)
}

var gf *Folder

func setup() {
	cbuf, err := ioutil.ReadFile(credpath)
	Ck(err)
	gf, err = NewFolder(cbuf, folderId, "mcp")
	Ck(err)
	return
}

func cleanup(t *testing.T) {
	// clean up from previous test
	gf.Clearcache()
	nodes, err := gf.AllNodes()
	Ck(err)
	for _, node := range nodes {
		if node.Num() >= 900 {
			err = gf.Rm(node.Name())
			Tassert(t, err == nil, Spf("%v: %v", node.Name(), err))
		}
	}
	gf.Clearcache()
}

func TestContent(t *testing.T) {
	cleanup(t)

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
