package web

import (
	"embed"
	"html/template"
	"log"
	"net/http"
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
	log.Printf("%v: %v", r.(error).Error(), msg)
}

type server struct {
	b *bot.Bot
	t *template.Template
}

func Serve(b *bot.Bot) (err error) {
	defer Return(&err)

	/*
		// save pid
		pidfn := Spf("/run/user/%d/docbot", os.Getuid())
		err = ioutil.WriteFile(pidfn, []byte(Spf("%d\n", os.Getpid())), 0644)
		Ck(err)
	*/

	s := &server{b: b}

	s.t, err = template.ParseFS(fs, "template/*")
	Ck(err)

	err = b.Init()
	Ck(err)

	// start server
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
	URL            string
	SearchQuery    string
	ResultsHeading string
}

// XXX deprecate
/*
func render(w http.ResponseWriter, name string, p *Page) {
	t, err := template.ParseFS(fs, Spf("template/%s.html", name))
	err = t.Execute(w, p)
	ckw(w, err)
}
*/

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
	if tmpl != "" {
		var node *google.Node
		node, err = tx.Opendoc(r, tmpl, ofn, s.b.Conf.Url, title)
		ckw(w, err)
		http.Redirect(w, r, node.URL(), http.StatusFound)
		return
	}

	p := &Page{URL: s.b.Conf.Url}
	// "01/02 03:04:05PM '06 -0700"
	p.YYYY = time.Now().Format("2006")

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

	p.NextNum, err = tx.NextNum()
	ckw(w, err)

	err = s.t.ExecuteTemplate(w, "index.html", p)
	ckw(w, err)

	return
}

func (s *server) search(w http.ResponseWriter, r *http.Request) {
	defer logw(r.URL)
	log.Println("search running")
	log.Println(r.URL)
	err := r.ParseForm()
	ckw(w, err)
	tx := s.b.StartTransaction()
	defer tx.Close()

	p := &Page{URL: s.b.Conf.Url}
	// "01/02 03:04:05PM '06 -0700"
	p.YYYY = time.Now().Format("2006")

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

	p.NextNum, err = tx.NextNum()
	ckw(w, err)

	err = s.t.ExecuteTemplate(w, "search.html", p)
	ckw(w, err)

	return
}
