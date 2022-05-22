package bot

import (
	"context"
	"embed"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"time"

	. "github.com/stevegt/goadapt"
	"google.golang.org/api/docs/v1"
	"google.golang.org/api/drive/v2"
	"google.golang.org/api/option"
)

func ckw(w http.ResponseWriter, err error) {
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

type Bot struct {
	Ls       bool
	Put      bool
	Serve    bool
	Credpath string
	Folderid string
	docs     *docs.Service
	drive    *drive.Service
	cache    *Cache
}

type Cache struct {
	nodes   []*Node
	time    time.Time
	nextNum int
}

type Node struct {
	file     *drive.File
	Name     string
	Id       string
	URL      string
	MimeType string
	Num      int
	Created  string
}

func (b *Bot) Init() (err error) {
	b.cache = &Cache{}

	cbuf, err := ioutil.ReadFile(b.Credpath)
	Ck(err)

	ctx := context.Background()

	b.docs, err = docs.NewService(ctx, option.WithCredentialsJSON(cbuf))
	Ck(err)

	b.drive, err = drive.NewService(ctx, option.WithCredentialsJSON(cbuf))
	Ck(err)
	return
}

func (b *Bot) Run() (res []byte, err error) {
	defer Return(&err)

	err = b.Init()

	switch true {
	case b.Ls:
		res, err = b.ls()
	case b.Put:
		b.put()
	case b.Serve:
		b.serve()
	default:
		Assert(false, "unhandled: %#v", b)
	}
	return
}

func (b *Bot) put() {
	parentref := &drive.ParentReference{Id: b.Folderid}

	title := Spf("foo")
	file, err := b.drive.Files.Insert(&drive.File{
		// OwnedByMe:       false, //service account can't use gdrive interface, that's why false
		CreatedDate:     time.Now().Format(time.RFC3339),
		MimeType:        "application/vnd.google-apps.document",
		Title:           title,
		WritersCanShare: false,
		Parents:         []*drive.ParentReference{parentref},
	}).Do()
	Ck(err)

	Pl(file.Id)

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
	Nodes          []*Node
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
	Pl("form", r.Form, val)
	return
}

func render(w http.ResponseWriter, name string, p *Page) {
	t, err := template.ParseFS(fs, Spf("template/%s.html", name))
	err = t.Execute(w, p)
	ckw(w, err)
}

func (b *Bot) index(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	ckw(w, err)

	p := &Page{}
	p.Nodes, err = b.getNodes()
	ckw(w, err)
	p.NextNum = b.cache.nextNum
	// "01/02 03:04:05PM '06 -0700"
	p.YYYY = time.Now().Format("2006")
	p.SearchQuery = parm(r, "query")

	render(w, "index", p)

	return
}

func (b *Bot) ls() (out []byte, err error) {
	defer Return(&err)
	nodes, err := b.getNodes()
	Ck(err)
	for _, n := range nodes {
		out = append(out, []byte(Spf("%s (%s) (%s)\n", n.Name, n.Id, n.MimeType))...)
	}
	return
}

func (b *Bot) getNodes() (nodes []*Node, err error) {
	defer Return(&err)

	if time.Now().Sub(b.cache.time) < time.Minute {
		return b.cache.nodes, nil
	}

	query := fmt.Sprintf("'%v' in parents", b.Folderid)

	var pageToken string
	for {

		q := b.drive.Files.List().Q(query)

		if pageToken != "" {
			q = q.PageToken(pageToken)
		}

		res, err := q.Do()
		Ck(err)

		re := regexp.MustCompile(`^mcp-(\d+)-`)
		for _, f := range res.Items {
			// f is a *drive.File
			m := re.FindStringSubmatch(f.Title)
			var num int
			if len(m) == 2 {
				num, _ = strconv.Atoi(m[1])
				if num > b.cache.nextNum {
					b.cache.nextNum = num + 1
				}
			}
			n := &Node{
				file:    f,
				Name:    f.Title,
				Num:     num,
				Created: f.CreatedDate,
				URL:     f.AlternateLink,
			}
			nodes = append(nodes, n)
		}

		pageToken = res.NextPageToken
		if pageToken == "" {
			break
		}
	}
	b.cache.nodes = nodes
	b.cache.time = time.Now()

	return b.cache.nodes, nil
}

/*
// return the next (unused) document number
func (b *Bot) NextNum() int {
  var files = getFiles();
  var nums = nodes.map(function(x) {return x.num});
  // Logger.log(nums);
  nums = nums.filter(isInteger);
  // Logger.log(nums);
  // Logger.log(max(nums));
  var next = max(nums) + 1;
  next = next < min_next ? min_next : next;
  return next;
}

*/
