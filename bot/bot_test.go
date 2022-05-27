package bot

import (
	"bytes"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	// "github.com/sergi/go-diff/diffmatchpatch"
	"github.com/sergi/go-diff/diffmatchpatch"
	. "github.com/stevegt/goadapt"
)

// regenerate testdata
const regen bool = true

const confpath = "testdata/docbot.conf"
const credpath = "../local/mcpbot-mcpbot-key.json"

func bot(t *testing.T) (b *Bot) {
	b = &Bot{
		Confpath: confpath,
		Credpath: credpath,
	}
	err := b.Init()
	Tassert(t, err == nil, err)
	return
}

func TestMkDoc(t *testing.T) {
	b := bot(t)

	fn := "mcp-910-test10"

	// clean up from previous fail
	_ = b.gf.Rm(fn)
	b.gf.Clearcache()

	r := &http.Request{}
	node, err := b.opendoc(r, b.Conf.Template, fn, "test 10")
	Tassert(t, err == nil, err)
	Tassert(t, node != nil)

	err = b.gf.Rm(fn)
	Tassert(t, err == nil, err)
	b.gf.Clearcache()

	ts := httptest.NewServer(http.HandlerFunc(b.index))
	defer ts.Close()
	res, err := http.Get(Spf("%s?filename=%s", ts.URL, fn))
	Tassert(t, err == nil, err)
	val, ok := res.Header["X-Auto-Login"]
	Tassert(t, ok, Spf("%#v", res))
	got := []byte(val[0])

	dmp := diffmatchpatch.New()

	reffn := "testdata/mkdoc.X-Auto-Login"
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
	b := bot(t)

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
	b := bot(t)

	fn := "mcp-4-why-numbered-docs"

	node, err := b.gf.Getnode(fn)
	Tassert(t, err == nil, err)
	Tassert(t, node != nil, Spf("%#v", node))
	Tassert(t, len(node.Id()) != 0, Spf("%#v", node))
}

func TestIndex(t *testing.T) {
	b := bot(t)

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
	b := bot(t)

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
	b := bot(t)

	// https://pkg.go.dev/net/http/httptest#NewRequest
	// https://golang.cafe/blog/golang-httptest-example.html
	ts := httptest.NewServer(http.HandlerFunc(b.index))
	defer ts.Close()
	res, err := http.Get(Spf("%s?filename=mcp-4-why-numbered-docs", ts.URL))
	Tassert(t, err == nil, err)
	val, ok := res.Header["X-Auto-Login"]
	Tassert(t, ok, Spf("%#v", res))
	got := []byte(val[0])

	dmp := diffmatchpatch.New()

	fn := "testdata/filename.X-Auto-Login"
	if regen {
		err = ioutil.WriteFile(fn, got, 0644)
		Ck(err)
	}
	ref, err := ioutil.ReadFile(fn)
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
