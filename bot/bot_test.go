package bot

import (
	"bytes"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"

	// "github.com/sergi/go-diff/diffmatchpatch"
	"github.com/sergi/go-diff/diffmatchpatch"
	. "github.com/stevegt/goadapt"
)

// regenerate testdata
const regen bool = false

const confpath = "testdata/docbot.conf"
const credpath = "../local/mcpbot-mcpbot-key.json"

func TestMain(m *testing.M) {
	setup()
	code := m.Run()
	// teardown()
	os.Exit(code)
}

var b *Bot

func setup() {
	b = &Bot{
		Confpath: confpath,
		Credpath: credpath,
	}
	err := b.Init()
	Ck(err)
	// cleanup(t, b)
	return
}

func cleanup(t *testing.T) {
	// clean up from previous test
	b.gf.Clearcache()
	nodes, err := b.gf.AllNodes()
	Ck(err)
	for _, node := range nodes {
		if node.Num() >= 900 {
			err = b.gf.Rm(node.Name())
			Tassert(t, err == nil, Spf("%v: %v", node.Name(), err))
		}
	}
	b.gf.Clearcache()
}

func TestMkDoc(t *testing.T) {
	cleanup(t)

	fn := "mcp-910-test10"
	title := "test 10"

	ts := httptest.NewServer(http.HandlerFunc(b.index))
	defer ts.Close()

	v := url.Values{}
	v.Set("filename", fn)
	v.Set("title", title)
	url := Spf("%s?%s", ts.URL, v.Encode())

	r, err := http.NewRequest("GET", url, nil)
	Tassert(t, err == nil, err)
	err = r.ParseForm()
	Tassert(t, err == nil, err)

	// test opendoc directly
	node, err := b.opendoc(r, b.Conf.Template, fn)
	Tassert(t, err == nil, err)
	Tassert(t, node != nil)

	// check title in body
	h, err := b.gf.GetHeaders(node)
	Tassert(t, err == nil, err)
	gotTitle, ok := h["Title"]
	Tassert(t, ok, Spf("%#v", h))
	// Pprint(h)
	Tassert(t, gotTitle == title, gotTitle)

	// clean up
	err = b.gf.Rm(fn)
	Tassert(t, err == nil, err)
	b.gf.Clearcache()

	// create via mock docbot server
	_, err = http.Get(url)
	Tassert(t, err == nil, err)

	// get document text
	node, err = b.gf.Getnode(fn)
	Tassert(t, err == nil, err)
	Tassert(t, node != nil, Spf("%#v", node))
	txt, err := b.gf.Doc2txt(node)
	Tassert(t, err == nil, err)
	got := []byte(txt)

	dmp := diffmatchpatch.New()

	reffn := "testdata/mkdoc.txt"
	if regen {
		err = ioutil.WriteFile(reffn, got, 0644)
		Ck(err)
	}
	ref, err := ioutil.ReadFile(reffn)
	Tassert(t, err == nil, err)
	diffs := dmp.DiffMain(string(ref), string(got), false)
	Tassert(t, bytes.Equal(ref, got), dmp.DiffPrettyText(diffs))
}

func TestMkSessionDoc(t *testing.T) {
	cleanup(t)

	fn := "mcp-911-test11"
	title := "test 11"
	date := "02 Jan 2006"
	speakers := "Alice Arms, Bob Barker, Carol Carnes"

	ts := httptest.NewServer(http.HandlerFunc(b.index))
	defer ts.Close()

	v := url.Values{}
	v.Set("title", title)
	v.Set("session_filename", fn)
	v.Set("session_date", date)
	v.Set("session_speakers", speakers)
	url := Spf("%s?%s", ts.URL, v.Encode())

	// create via mock docbot server
	_, err := http.Get(url)
	Tassert(t, err == nil, err)

	// Pprint(url)
	// Pprint(res.Status)
	// Pprint(res.Header)

	// get document text
	node, err := b.gf.Getnode(fn)
	Tassert(t, err == nil, err)
	Tassert(t, node != nil, Spf("%#v", node))
	txt, err := b.gf.Doc2txt(node)
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

	err = b.gf.Rm(fn)
	Tassert(t, err == nil, err)
}

func TestLs(t *testing.T) {
	cleanup(t)

	got, err := b.ls()
	Tassert(t, err == nil, err)
	// Pl(string(got))

	reffn := "testdata/ls.txt"
	if regen {
		err = ioutil.WriteFile(reffn, got, 0644)
		Ck(err)
	}

	ref, err := ioutil.ReadFile(reffn)
	Tassert(t, err == nil, err)

	dmp := diffmatchpatch.New()
	diffs := dmp.DiffMain(string(ref), string(got), false)
	Tassert(t, bytes.Equal(ref, got), dmp.DiffPrettyText(diffs))
}

func TestGetnode(t *testing.T) {
	cleanup(t)

	fn := "mcp-4-why-numbered-docs"

	node, err := b.gf.Getnode(fn)
	Tassert(t, err == nil, err)
	Tassert(t, node != nil, Spf("%#v", node))
	Tassert(t, len(node.Id()) != 0, Spf("%#v", node))
}

func TestIndex(t *testing.T) {
	cleanup(t)

	// https://pkg.go.dev/net/http/httptest#NewRequest
	// https://golang.cafe/blog/golang-httptest-example.html
	ts := httptest.NewServer(http.HandlerFunc(b.index))
	defer ts.Close()
	res, err := http.Get(ts.URL)
	Tassert(t, err == nil, err)
	got, err := io.ReadAll(res.Body)
	Tassert(t, err == nil, err)
	res.Body.Close()

	reffn := "testdata/index.html"
	if regen {
		err = ioutil.WriteFile(reffn, got, 0644)
		Ck(err)
	}

	ref, err := ioutil.ReadFile(reffn)
	Tassert(t, err == nil, err)

	dmp := diffmatchpatch.New()
	diffs := dmp.DiffMain(string(ref), string(got), false)
	Tassert(t, bytes.Equal(ref, got), dmp.DiffPrettyText(diffs))
}

func TestSearch(t *testing.T) {
	cleanup(t)

	// https://pkg.go.dev/net/http/httptest#NewRequest
	// https://golang.cafe/blog/golang-httptest-example.html
	ts := httptest.NewServer(http.HandlerFunc(b.index))
	defer ts.Close()
	res, err := http.Get(Spf("%s?query=maker", ts.URL))
	Tassert(t, err == nil, err)
	got, err := io.ReadAll(res.Body)
	Tassert(t, err == nil, err)
	res.Body.Close()

	reffn := "testdata/search.html"
	if regen {
		err = ioutil.WriteFile(reffn, got, 0644)
		Ck(err)
	}

	ref, err := ioutil.ReadFile(reffn)
	Tassert(t, err == nil, err)

	dmp := diffmatchpatch.New()
	diffs := dmp.DiffMain(string(ref), string(got), false)
	Tassert(t, bytes.Equal(ref, got), dmp.DiffPrettyText(diffs))
}

func TestFilename(t *testing.T) {
	cleanup(t)

	fn := "mcp-1-repository-github"

	// https://pkg.go.dev/net/http/httptest#NewRequest
	// https://golang.cafe/blog/golang-httptest-example.html
	ts := httptest.NewServer(http.HandlerFunc(b.index))
	defer ts.Close()

	v := url.Values{}
	v.Set("filename", fn)
	url := Spf("%s?%s", ts.URL, v.Encode())

	_, err := http.Get(url)
	Tassert(t, err == nil, err)

	// get document text
	node, err := b.gf.Getnode(fn)
	Tassert(t, err == nil, err)
	Tassert(t, node != nil, Spf("%#v", node))
	txt, err := b.gf.Doc2txt(node)
	Tassert(t, err == nil, err)
	got := []byte(txt)

	dmp := diffmatchpatch.New()

	reffn := "testdata/filename.txt"
	if regen {
		err = ioutil.WriteFile(reffn, got, 0644)
		Ck(err)
	}
	ref, err := ioutil.ReadFile(reffn)
	Tassert(t, err == nil, err)
	diffs := dmp.DiffMain(string(ref), string(got), false)
	Tassert(t, bytes.Equal(ref, got), dmp.DiffPrettyText(diffs))
}

func TestText(t *testing.T) {
	cleanup(t)

	fn := "mcp-4-why-numbered-docs"

	node, err := b.gf.Getnode(fn)
	Tassert(t, err == nil, err)
	Tassert(t, node != nil, Spf("%#v", node))

	txt, err := b.gf.Doc2txt(node)
	Tassert(t, err == nil, err)
	got := []byte(txt)

	dmp := diffmatchpatch.New()

	reffn := "testdata/doc2txt.txt"
	if regen {
		err = ioutil.WriteFile(reffn, []byte(got), 0644)
		Ck(err)
	}
	ref, err := ioutil.ReadFile(reffn)
	Tassert(t, err == nil, err)
	diffs := dmp.DiffMain(string(ref), string(got), false)
	Tassert(t, bytes.Equal(ref, got), dmp.DiffPrettyText(diffs))

}

/*
// XXX this returns the header and html from gdocs as the result of
// the redirect
func TestFilename(t *testing.T) {
	b := &Bot{
		Conf: &Conf{Credpath: credpath, Folderid: folderId},
	}
	err := b.Init()
	Tassert(t, err == nil, err)

	// https://pkg.go.dev/net/http/httptest#NewRequest
	// https://golang.cafe/blog/golang-httptest-example.html
	ts := httptest.NewServer(http.HandlerFunc(b.index))
	defer ts.Close()
	res, err := http.Get(Spf("%s?filename=mcp-4-why-numbered-docs", ts.URL))
	Tassert(t, err == nil, err)
	head := []byte(Spf("%s", res.Header))
	body, err := io.ReadAll(res.Body)
	Tassert(t, err == nil, err)
	res.Body.Close()

	dmp := diffmatchpatch.New()

	hfn := "testdata/filename.head"
	if regen {
		err = ioutil.WriteFile(hfn, head, 0644)
		Ck(err)
	}
	href, err := ioutil.ReadFile(hfn)
	Tassert(t, err == nil, err)
	diffs := dmp.DiffMain(string(href), string(head), false)
	Tassert(t, bytes.Equal(href, head), dmp.DiffPrettyText(diffs))

	bfn := "testdata/filename.html"
	if regen {
		err = ioutil.WriteFile(bfn, body, 0644)
		Ck(err)
	}
	bref, err := ioutil.ReadFile(bfn)
	Tassert(t, err == nil, err)
	diffs = dmp.DiffMain(string(bref), string(body), false)
	Tassert(t, bytes.Equal(bref, body), dmp.DiffPrettyText(diffs))
}
*/
