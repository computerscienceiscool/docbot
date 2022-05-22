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

const credpath = "../local/mcpbot-mcpbot-key.json"
const folderId = "1HcCIw7ppJZPD9GEHccnkgNYUwhAGCif6"

func TestLs(t *testing.T) {
	b := &Bot{
		Ls:   true,
		Conf: &Conf{Credpath: credpath, Folderid: folderId},
	}
	err := b.Init()
	Tassert(t, err == nil, err)

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
	b := &Bot{
		Conf: &Conf{Credpath: credpath, Folderid: folderId},
	}
	err := b.Init()
	Tassert(t, err == nil, err)

	// https://pkg.go.dev/net/http/httptest#NewRequest
	// https://golang.cafe/blog/golang-httptest-example.html
	ts := httptest.NewServer(http.HandlerFunc(b.index))
	defer ts.Close()
	res, err := http.Get(ts.URL)
	Tassert(t, err == nil, err)
	got, err := io.ReadAll(res.Body)
	Tassert(t, err == nil, err)
	res.Body.Close()

	Pl(string(got))

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
