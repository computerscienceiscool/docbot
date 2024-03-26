package bot

import (
	"encoding/json"
	"io/ioutil"
	"regexp"

	"github.com/stevegt/docbot/google"
	"github.com/stevegt/docbot/transaction"
	. "github.com/stevegt/goadapt"
)

type Doc interface {
	Created() string
	Id() string
	MimeType() string
	Name() string
	Num() int
	URL() string
}

type Conf struct {
	Folderid        string
	Docprefix       string
	Template        string
	SessionTemplate string `json:"session_template"`
	CSWGTemplate    string `json:"cswg_template"`
	Url             string
	Listen          string
	MinNextNum      int
}

type Bot struct {
	Ls         bool
	Serve      bool
	Confpath   string
	Credpath   string
	Conf       *Conf
	repo       *google.Folder
	docpattern *regexp.Regexp
}

func (b *Bot) Init() (err error) {

	err = b.LoadConf(b.Confpath)
	Ck(err)

	cbuf, err := ioutil.ReadFile(b.Credpath)
	Ck(err)

	pat := Spf("^%s-(\\d+)-", b.Conf.Docprefix)
	b.docpattern, err = regexp.Compile(pat)
	Ck(err)

	b.repo, err = google.NewFolder(cbuf, b.Conf.Folderid, b.Conf.Docprefix, b.Conf.MinNextNum)
	Ck(err)

	return
}

func (b *Bot) LoadConf(fn string) (err error) {
	defer Return(&err)
	buf, err := ioutil.ReadFile(fn)
	Ck(err)
	conf := &Conf{}
	err = json.Unmarshal(buf, conf)
	Ck(err)
	b.Conf = conf
	return
}

func (b *Bot) StartTransaction() (tx *transaction.Transaction) {
	tx = transaction.Start(b.repo)
	return
}

// XXX

/*
// get document text
func (b *Bot) getText(fn string) (txt string, err error) {
	defer Return(&err)
	tx := b.repo.StartTransaction()
	defer tx.Close()
	node, err := tx.Getnode(fn)
	Ck(err)
	Assert(node != nil, fn)
	txt, err = tx.Doc2txt(node)
	Ck(err)
	return
}

func (b *Bot) getJson(fn string) (buf []byte, err error) {
	defer Return(&err)
	tx := b.repo.StartTransaction()
	defer tx.Close()
	node, err := tx.Getnode(fn)
	Ck(err)
	Assert(node != nil, fn)
	buf, err = tx.Doc2json(node)
	Ck(err)
	return
}
*/
