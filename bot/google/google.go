package google

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"sync"
	"time"

	. "github.com/stevegt/goadapt"
	"google.golang.org/api/docs/v1"
	"google.golang.org/api/drive/v2"
	"google.golang.org/api/option"
)

type Google struct {
	folderid string
	mu       sync.Mutex
	docs     *docs.Service
	drive    *drive.Service
	cache    *cache
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

type cache struct {
	nodes   []*Node
	time    time.Time
	lastNum int
}

// New returns a Google object that represents a single gdrive folder.
// We assume that the folder is accessible by the service account
// json credentials provided in cbuf.
func New(cbuf []byte, folderid string) (g *Google, err error) {
	defer Return(&err)

	g = &Google{folderid: folderid}
	g.cache = &cache{}

	ctx := context.Background()

	g.docs, err = docs.NewService(ctx, option.WithCredentialsJSON(cbuf))
	Ck(err)

	g.drive, err = drive.NewService(ctx, option.WithCredentialsJSON(cbuf))
	Ck(err)
	return
}

func (g *Google) put() {

	parentref := &drive.ParentReference{Id: g.folderid}

	title := Spf("foo")
	file, err := g.drive.Files.Insert(&drive.File{
		// OwnedByMe:       false, //service account can't use gdrive interface, that's why false
		CreatedDate:     time.Now().Format(time.RFC3339),
		MimeType:        "application/vnd.google-apps.document",
		Title:           title,
		WritersCanShare: false,
		Parents:         []*drive.ParentReference{parentref},
	}).Do()
	Ck(err)

	Pl(file.Id)

	// XXX clear cache
}

/*
// GetNodes takes a query string and returns all matching nodes.  If
// the query string is empty, then we cache the results.
func (g *Google) GetNodes(query string) (nodes []*Node, err error) {
	defer Return(&err)

	g.mu.Lock()
	defer g.mu.Unlock()

	// expire cache after 1 minute
	if time.Now().Sub(g.cache.time) > time.Minute {
		g.cache = &Nodes{}
	}

	if query == "" {
		query = fmt.Sprintf("'%v' in parents", g.folderid)
	}

	var pageToken string
	var lastNum int
	for {

		q := g.drive.Files.List().Q(query)

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
				if num > lastNum {
					lastNum = num
				}
			}
			n := &Node{
				file:    f,
				Name:    f.Title,
				Num:     num,
				Created: f.CreatedDate,
				URL:     f.AlternateLink,
			}
			nodes.nodes = append(nodes.nodes, n)
		}

		pageToken = res.NextPageToken
		if pageToken == "" {
			break
		}
	}

	if query == "" {
		g.cache.nodes = nodes
		g.cache.lastNum = lastNum
		g.cache.time = time.Now()
	}

	return nodes, nil
}
*/

// FindNodes takes a query string and returns all matching nodes.
func (g *Google) FindNodes(query string) (nodes []*Node, err error) {
	defer Return(&err)
	nodes, _, err = g.getNodes(query)
	Ck(err)
	return
}

// AllNodes returns all nodes and caches the results.
func (g *Google) AllNodes() (nodes []*Node, err error) {
	defer Return(&err)

	g.mu.Lock()
	defer g.mu.Unlock()

	if time.Now().Sub(g.cache.time) < time.Minute {
		return g.cache.nodes, nil
	}

	nodes, lastNum, err := g.getNodes("")
	Ck(err)

	g.cache.nodes = nodes
	g.cache.lastNum = lastNum
	g.cache.time = time.Now()

	return
}

func (g *Google) getNodes(query string) (nodes []*Node, lastNum int, err error) {
	defer Return(&err)

	if query == "" {
		query = fmt.Sprintf("'%v' in parents", g.folderid)
	} else {
		query = fmt.Sprintf("'%v' in parents and %s", g.folderid, query)
	}

	var pageToken string
	for {

		q := g.drive.Files.List().Q(query)

		if pageToken != "" {
			q = q.PageToken(pageToken)
		}

		res, err := q.Do()
		Ck(err)

		// XXX s/mcp/g.docPrefix/g
		re := regexp.MustCompile(`^mcp-(\d+)-`)
		for _, f := range res.Items {
			// f is a *drive.File
			m := re.FindStringSubmatch(f.Title)
			var num int
			if len(m) == 2 {
				num, _ = strconv.Atoi(m[1])
				if num > lastNum {
					lastNum = num
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

	return
}

// return the last used document number
func (g *Google) LastNum() (last int, err error) {
	defer Return(&err)

	// refresh cache
	_, err = g.AllNodes()
	Ck(err)

	last = g.cache.lastNum
	return
}
