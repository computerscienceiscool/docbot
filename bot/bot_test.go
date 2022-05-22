package bot

import (
	"bytes"
	"io/ioutil"
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
		Ls:       true,
		Credpath: credpath,
		Folderid: folderId,
	}
	err := b.Init()
	Tassert(t, err == nil, err)

	got := b.ls()
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

func TestIndexHtml(t *testing.T) {
	b := &Bot{
		Credpath: credpath,
		Folderid: folderId,
	}
	err := b.Init()
	Tassert(t, err == nil, err)

	got := b.indexHtml()
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
