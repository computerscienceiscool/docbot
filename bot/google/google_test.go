package google

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	// "github.com/sergi/go-diff/diffmatchpatch"

	. "github.com/stevegt/goadapt"
)

// regenerate testdata
const regen bool = false

const (
	credpath = "../../local/mcpbot-mcpbot-key.json"
	folderId = "1HcCIw7ppJZPD9GEHccnkgNYUwhAGCif6"
	template = "mcp-template"
)

/*
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
*/

func setup(t *testing.T) (tx *transaction) {
	cbuf, err := ioutil.ReadFile(credpath)
	Tassert(t, err == nil, err)

	gf, err := NewFolder(cbuf, folderId, "mcp", 900)
	Tassert(t, err == nil, err)

	tx = gf.StartTransaction()

	// clean up from previous test
	nodes, err := tx.AllNodes()
	Tassert(t, err == nil, err)
	for _, node := range nodes {
		if node.Num() >= 900 {
			err = tx.Rm(node)
			if err != nil {
				Pf("%v: %v\n", node.Name(), err)
			}
		}
	}
	time.Sleep(time.Second)
	return
}

func waitfor(tx *transaction, node *Node) {
	for i := 0; i < 10; i++ {
		doc, err := tx.gf.docs.Documents.Get(node.id).Do()
		if doc != nil && err == nil {
			break
		}
		Pf("waitfor: %v: %v\n", node.name, err)
		time.Sleep(time.Second)
	}
}

func TestOpenDoc(t *testing.T) {
	tx := setup(t)
	defer tx.Close()

	fn := "mcp-910-test10"
	title := "test 10"
	baseUrl := "http://example.com"
	v := url.Values{}
	v.Set("filename", fn)
	v.Set("title", title)
	url := Spf("%s?%s", "/", v.Encode())

	// create
	r, err := http.NewRequest("GET", url, nil)
	Tassert(t, err == nil, err)
	err = r.ParseForm()
	Tassert(t, err == nil, err)
	node, err := tx.Opendoc(r, template, fn, baseUrl)
	Tassert(t, err == nil, err)
	Tassert(t, node != nil)

	// check title in body
	h, err := tx.GetHeaders(node)
	Tassert(t, err == nil, err)
	gotTitle, ok := h["Title"]
	Tassert(t, ok, Spf("%#v", h))
	// Pprint(h)
	Tassert(t, gotTitle == title, gotTitle)

}

func TestContent(t *testing.T) {
	tx := setup(t)
	defer tx.Close()

	fn := "mcp-4-why-numbered-docs"
	expect := "Name: mcp-4-why-numbered-docs\n"

	node, err := tx.Getnode(fn)
	Tassert(t, err == nil, err)
	Tassert(t, node != nil, Spf("%#v", node))

	// https://github.com/rsbh/doc2md/blob/a740060638ca55813c25c7e4a6cf7774e3cbd63f/pkg/transformer/doc2json.go#L368
	doc, err := tx.gf.docs.Documents.Get(node.Id()).Do()
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

func TestFindText(t *testing.T) {
	tx := setup(t)
	defer tx.Close()

	fn := "session-template"

	node, err := tx.Getnode(fn)
	Tassert(t, err == nil, err)
	Tassert(t, node != nil, Spf("%#v", node))

	el, err := tx.FindTextRun(node, "UNLOCK_URL")
	Tassert(t, err == nil, err)
	Tassert(t, el != nil)
	// Pprint(el)
	Tassert(t, el.TextRun.Content == "UNLOCK_URL", el)
	Tassert(t, el.TextRun.TextStyle.Link.Url == "http://example.com", el)
}

/*
	XXX

	// https://github.com/rsbh/doc2md/blob/a740060638ca55813c25c7e4a6cf7774e3cbd63f/pkg/transformer/doc2json.go#L368
	doc, err := tx.gf.docs.Documents.Get(node.Id()).Do()
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
	err = ioutil.WriteFile("/tmp/mcp-912.json", buf, 0644)
	Tassert(t, err == nil, err)



	// create via mock docbot server
	_, err := http.Get(url)
	Tassert(t, err == nil, err)

	// Pprint(url)
	// Pprint(res.Status)
	// Pprint(res.Header)

	// get document text
	txt, err := b.getText(fn)
	Tassert(t, err == nil, err)
	got := []byte(txt)

	dmp := diffmatchpatch.New()

	reffn := "testdata/mksessiondoc.txt"
	if regen {
		err = ioutil.WriteFile(reffn, got, 0644)
		Ck(err)
	}
	ref, err := ioutil.ReadFile(reffn)
	Tassert(t, err == nil, err)
	diffs := dmp.DiffMain(string(ref), string(got), false)
	Tassert(t, bytes.Equal(ref, got), dmp.DiffPrettyText(diffs))

	// save a copy of content for reverse engineering
	buf, err := b.getJson(fn)
	Tassert(t, err == nil, err)
	err = ioutil.WriteFile("/tmp/mcp-911.json", buf, 0644)
	Tassert(t, err == nil, err)
*/
