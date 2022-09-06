package web

import (
	"embed"
	"html/template"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/stevegt/docbot/bot"
	"github.com/stevegt/docbot/google"
	. "github.com/stevegt/goadapt"
)

//go:embed template/*
var fs embed.FS

func ckw(w http.ResponseWriter, err error) {
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		panic(err)
	}
}

func logw(args ...interface{}) {
	r := recover()
	if r == nil {
		return
	}
	msg := FormatArgs(args...)
	log.Printf("error: %v: %v", r.(error).Error(), msg)
}

type server struct {
	b         *bot.Bot
	t         *template.Template
	searchUrl string
}

func Serve(b *bot.Bot) (err error) {
	defer Return(&err)

	/*
		// save pid
		pidfn := Spf("/run/user/%d/docbot", os.Getuid())
		err = ioutil.WriteFile(pidfn, []byte(Spf("%d\n", os.Getpid())), 0644)
		Ck(err)
	*/

	err = b.Init()
	Ck(err)

	s := &server{b: b}

	s.t, err = template.ParseFS(fs, "template/*")
	Ck(err)

	s.searchUrl = Spf("%s/search", s.b.Conf.Url)

	// start server
	http.HandleFunc("/doc/", s.doc)
	http.HandleFunc("/unlock/", s.unlock)
	http.HandleFunc("/search", s.search)
	http.HandleFunc("/", s.index)
	listen := b.Conf.Listen
	if listen == "" {
		listen = ":80"
	}
	err = http.ListenAndServe(listen, nil)
	Ck(err)
	return
}

type Page struct {
	Nodes          []*google.Node
	YYYY           string
	NextNum        int
	BaseURL        string
	PageURL        string
	UnlockBase     string
	SearchURL      string
	SearchQuery    string
	ResultsHeading string
}

func newPage(s *server, uri string, nextnum int) (p *Page) {
	p = &Page{
		NextNum:    nextnum,
		BaseURL:    s.b.Conf.Url,
		PageURL:    Spf("%s%s", s.b.Conf.Url, uri),
		SearchURL:  s.searchUrl,
		UnlockBase: Spf("%s/unlock", s.b.Conf.Url),
		// "01/02 03:04:05PM '06 -0700"
		YYYY: time.Now().Format("2006"),
	}
	return p
}

func (s *server) index(w http.ResponseWriter, r *http.Request) {
	defer logw(r.URL)
	log.Println(r.URL)
	err := r.ParseForm()
	ckw(w, err)
	tx := s.b.StartTransaction()
	defer tx.Close()

	// create doc and redirect
	var tmpl string
	var ofn string
	var title string
	fn := r.Form.Get("filename")
	sfn := r.Form.Get("session_filename")
	if fn != "" {
		ofn = fn
		title = r.Form.Get("title")
		tmpl = s.b.Conf.Template
	} else if sfn != "" {
		ofn = sfn
		title = r.Form.Get("session_title")
		tmpl = s.b.Conf.SessionTemplate
	}

	nextNum, err := tx.NextNum()
	ckw(w, err)
	p := newPage(s, "/", nextNum)

	unlockPrefix := Spf("%s/%s", p.UnlockBase, s.b.Conf.Docprefix)

	if tmpl != "" {
		var node *google.Node
		node, err = tx.OpenCreate(r, tmpl, ofn, unlockPrefix, title)
		ckw(w, err)
		err = tx.Unlock(node)
		ckw(w, err)
		http.Redirect(w, r, node.URL(), http.StatusFound)
		return
	}

	err = s.t.ExecuteTemplate(w, "index.html", p)
	ckw(w, err)

	return
}

func (s *server) search(w http.ResponseWriter, r *http.Request) {
	defer logw(r.URL)
	log.Println(r.URL)
	err := r.ParseForm()
	ckw(w, err)
	tx := s.b.StartTransaction()
	defer tx.Close()

	nextNum, err := tx.NextNum()
	ckw(w, err)
	p := newPage(s, "/search", nextNum)

	p.SearchQuery = r.Form.Get("query")
	if p.SearchQuery == "" {
		p.Nodes, err = tx.AllNodes()
		ckw(w, err)
		p.ResultsHeading = "All documents:"
	} else {
		q := Spf("fullText contains '%s'", p.SearchQuery)
		p.Nodes, err = tx.FindNodes(q)
		ckw(w, err)
		p.ResultsHeading = Spf("Search results for '%s':", p.SearchQuery)
	}

	err = s.t.ExecuteTemplate(w, "search.html", p)
	ckw(w, err)

	return
}

func (s *server) unlock(w http.ResponseWriter, r *http.Request) {
	defer logw(r.URL)
	log.Println(r.URL)
	err := r.ParseForm()
	ckw(w, err)
	tx := s.b.StartTransaction()
	defer tx.Close()

	nextNum, err := tx.NextNum()
	ckw(w, err)
	p := newPage(s, "/unlock", nextNum)
	_ = p

	parts := strings.Split(r.URL.String(), "/")
	// first part is empty because of leading slash
	if len(parts) != 3 {
		log.Printf("error: wrong parts count: %s: %v", r.URL, parts)
		http.Redirect(w, r, s.searchUrl, http.StatusFound)
		return
	}

	// XXX move this to tx so we can unlock from cli
	prefix := Spf("%s-", parts[2])
	node, err := tx.OpenPrefix(prefix)
	ckw(w, err)
	if node == nil {
		log.Printf("error: doc not found: %s", prefix)
		http.Redirect(w, r, s.searchUrl, http.StatusFound)
		return
	}

	err = tx.Unlock(node)
	ckw(w, err)

	http.Redirect(w, r, node.URL(), http.StatusFound)
	return
}

func (s *server) doc(w http.ResponseWriter, r *http.Request) {
	defer logw(r.URL)
	log.Println(r.URL)
	err := r.ParseForm()
	ckw(w, err)
	tx := s.b.StartTransaction()
	defer tx.Close()

	parts := strings.Split(r.URL.String(), "/")
	// first part is empty because of leading slash
	if len(parts) != 3 {
		log.Printf("error: wrong parts count: %s: %v", r.URL, parts)
		http.Redirect(w, r, s.searchUrl, http.StatusFound)
		return
	}

	// XXX move this to tx so we can get google doc url via cli
	prefix := Spf("%s-", parts[2])
	node, err := tx.OpenPrefix(prefix)
	ckw(w, err)
	if node == nil {
		log.Printf("error: doc not found: %s", prefix)
		http.Redirect(w, r, s.searchUrl, http.StatusFound)
		return
	}

	http.Redirect(w, r, node.URL(), http.StatusFound)
	return
}
