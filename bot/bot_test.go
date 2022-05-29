package bot

import (
	"bytes"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	// "github.com/sergi/go-diff/diffmatchpatch"
	"github.com/sergi/go-diff/diffmatchpatch"
	. "github.com/stevegt/goadapt"
)

// regenerate testdata
const regen bool = false

const confpath = "testdata/docbot.conf"
const credpath = "../local/mcpbot-mcpbot-key.json"

/*
func TestMain(m *testing.M) {
	setup()
	code := m.Run()
	// teardown()
	os.Exit(code)
}
*/

// var b *Bot

func setup(t *testing.T) (b *Bot) {
	b = &Bot{
		Confpath: confpath,
		Credpath: credpath,
	}
	err := b.Init()
	Tassert(t, err == nil, err)
	cleanup(t, b)
	return
}

// clean up from previous test
func cleanup(t *testing.T, b *Bot) {
	tx := b.repo.StartTransaction()
	defer tx.Close()
	nodes, err := tx.AllNodes()
	Tassert(t, err == nil, err)
	for _, node := range nodes {
		if node.Num() >= 900 {
			err = tx.Rm(node.Name())
			Tassert(t, err == nil, Spf("%v: %v", node.Name(), err))
		}
	}
}

func TestMkDoc(t *testing.T) {
	b := setup(t)

	ts := httptest.NewServer(http.HandlerFunc(b.index))
	defer ts.Close()

	fn := "mcp-910-test10"
	title := "test 10"
	v := url.Values{}
	v.Set("filename", fn)
	v.Set("title", title)
	url := Spf("%s?%s", ts.URL, v.Encode())

	// create via mock docbot server
	_, err := http.Get(url)
	Tassert(t, err == nil, err)

	// get document text
	txt, err := b.getText(fn)
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
	b := setup(t)

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

}

func TestLs(t *testing.T) {
	b := setup(t)

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

func TestIndex(t *testing.T) {
	b := setup(t)

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
	b := setup(t)

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
	b := setup(t)

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
	txt, err := b.getText(fn)
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
	b := setup(t)

	fn := "mcp-4-why-numbered-docs"

	// get document text
	txt, err := b.getText(fn)
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
