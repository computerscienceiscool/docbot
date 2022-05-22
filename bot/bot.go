package bot

import (
	"embed"
	"encoding/json"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/stevegt/docbot/bot/google"
	"github.com/stevegt/envi"
	. "github.com/stevegt/goadapt"
)

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

type Bot struct {
	Ls    bool
	Put   bool
	Serve bool
	Conf  *Conf
	g     *google.Google
}

type Conf struct {
	Credpath   string
	Folderid   string
	Url        string
	MinNextNum int
}

func (b *Bot) Init() (err error) {

	cbuf, err := ioutil.ReadFile(b.Conf.Credpath)
	Ck(err)

	b.g, err = google.New(cbuf, b.Conf.Folderid)
	Ck(err)

	return
}

func (b *Bot) LoadConf() (err error) {
	defer Return(&err)
	fn := envi.String("DOCBOT_CONF", ".docbot.conf")
	buf, err := ioutil.ReadFile(fn)
	Ck(err)
	conf := &Conf{}
	err = json.Unmarshal(buf, conf)
	Ck(err)
	b.Conf = conf
	return
}

func (b *Bot) Run() (res []byte, err error) {
	defer Return(&err)

	err = b.LoadConf()
	Ck(err)
	err = b.Init()
	Ck(err)

	switch true {
	case b.Ls:
		res, err = b.ls()
	case b.Put:
		// b.put()
	case b.Serve:
		b.serve()
	default:
		Assert(false, "unhandled: %#v", b)
	}
	return
}

//go:embed template/*
var fs embed.FS

func (b *Bot) serve() {
	// save pid
	pidfn := Spf("/run/user/%d/docbot", os.Getuid())
	err := ioutil.WriteFile(pidfn, []byte(Spf("%d\n", os.Getpid())), 0644)
	Ck(err)

	// start server
	http.HandleFunc("/", b.index)
	log.Fatal(http.ListenAndServe(":8080", nil))
}

type Page struct {
	Nodes          []*google.Node
	YYYY           string
	NextNum        int
	URL            string
	SearchQuery    string
	ResultsHeading string
}

func parm(r *http.Request, key string) (val string) {
	if len(r.Form[key]) > 0 {
		val = r.Form[key][0]
	}
	Pl("form", key, val, r.Form)
	return
}

func render(w http.ResponseWriter, name string, p *Page) {
	t, err := template.ParseFS(fs, Spf("template/%s.html", name))
	err = t.Execute(w, p)
	ckw(w, err)
}

func (b *Bot) index(w http.ResponseWriter, r *http.Request) {
	defer logw(r.URL)

	err := r.ParseForm()
	ckw(w, err)

	p := &Page{}
	p.URL = b.Conf.Url
	// "01/02 03:04:05PM '06 -0700"
	p.YYYY = time.Now().Format("2006")

	p.SearchQuery = parm(r, "query")
	if p.SearchQuery == "" {
		p.Nodes, err = b.g.AllNodes()
		ckw(w, err)
		p.ResultsHeading = ""
	} else {
		q := Spf("fullText contains '%s'", p.SearchQuery)
		p.Nodes, err = b.g.FindNodes(q)
		ckw(w, err)
		p.ResultsHeading = Spf("Search results for '%s':", p.SearchQuery)
	}

	p.NextNum, err = b.NextNum()
	ckw(w, err)

	render(w, "index", p)

	return
}

func (b *Bot) ls() (out []byte, err error) {
	defer Return(&err)
	nodes, err := b.g.AllNodes()
	Ck(err)
	for _, n := range nodes {
		out = append(out, []byte(Spf("%s (%s) (%s)\n", n.Name, n.Id, n.MimeType))...)
	}
	return
}

// return the next (unused) document number
func (b *Bot) NextNum() (next int, err error) {
	defer Return(&err)
	last, err := b.g.LastNum()
	Ck(err)
	next = last + 1
	if next < b.Conf.MinNextNum {
		next = b.Conf.MinNextNum
	}
	// XXX race condition -- check to see if doc exists
	return
}

/*
// respond to a GET request
function doGet(e) {
  // e.parameter contains the GET args
  var parms = e.parameter;
  // session_filename or filename is a file to be created and/or opened
  // var session_filename = e.parameter.session_filename;
  // var filename = e.parameter.filename;
  // query is a string containing search keywords
  var query = parms.query;
  var self_url = ScriptApp.getService().getUrl();
  if (parms.unlock) {
	XXX
    var node = unlock(parms.filename)
    return HtmlService.createHtmlOutput("<script>window.top.location.href='" + node.url + "';</script>");
  } else if (parms.session_filename) {
	XXX
    var node = opendoc('session-template', parms.session_filename, parms.session_title, parms)
    // return HTML that opens the file in google docs
    return HtmlService.createHtmlOutput("<script>window.top.location.href='" + node.url + "';</script>");
  } else if (parms.filename) {
	XXX
    var node = opendoc('mcp-template', parms.filename, parms.title, parms)
    // return HTML that opens the file in google docs
    return HtmlService.createHtmlOutput("<script>window.top.location.href='" + node.url + "';</script>");
  }

  // load persistent db content
  // var db = PropertiesService.getScriptProperties();

  // build the index page from template
  var tmpl = HtmlService.createTemplateFromFile('index');
  var nodes;
  if (query) {
	XXX
    nodes = searchnodes("fullText contains '" + query + "'");
    tmpl.results_heading = "Search results for '" + query + "':";
  } else {
    nodes = getnodes();
    query = '';
    tmpl.results_heading = "All documents:";
  }
  // provide content for the <?= foo ?> variables in the template
  // https://developers.google.com/apps-script/guides/html/templates
  tmpl.nodes = nodes;
  tmpl.self_url = self_url;
  tmpl.query = query;
  tmpl.next_num = next_num();
  tmpl.yyyy = now.getFullYear()


  // return the index page
  return tmpl.evaluate();
}
*/
