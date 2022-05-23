package google

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"time"

	"github.com/stevegt/docbot/bot/google/cache"
	. "github.com/stevegt/goadapt"
	"google.golang.org/api/docs/v1"
	"google.golang.org/api/drive/v2"
	"google.golang.org/api/option"
)

type Google struct {
	folderid string
	docs     *docs.Service
	drive    *drive.Service
	cache    *cache.Cache
}

func (g *Google) Copy(fileid, newName string) (newNode *cache.Node, err error) {
	defer Return(&err)
	parentref := &drive.ParentReference{Id: g.folderid}
	file := &drive.File{Parents: []*drive.ParentReference{parentref}, Title: newName}
	f, err := g.drive.Files.Copy(fileid, file).Do()
	Ck(err)
	newNode, err = g.NewNode(f)
	Ck(err)
	return
}

// New returns a Google object that represents a single gdrive folder.
// We assume that the folder is accessible by the service account
// json credentials provided in cbuf.
func New(cbuf []byte, folderid string) (g *Google, err error) {
	defer Return(&err)

	g = &Google{folderid: folderid}
	g.cache = cache.NewCache()

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

}

func (g *Google) getNodes(query string) (nodes []*cache.Node, lastNum int, err error) {
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

		for _, f := range res.Items {
			// f is a *drive.File
			_, err := g.NewNode(f)
			Ck(err)
		}

		pageToken = res.NextPageToken
		if pageToken == "" {
			break
		}
	}

	return
}

// FindNodes takes a query string and returns all matching nodes.
func (g *Google) FindNodes(query string) (nodes []*cache.Node, err error) {
	defer Return(&err)
	nodes, _, err = g.getNodes(query)
	Ck(err)
	return
}

// AllNodes returns all nodes and caches the results.
func (g *Google) AllNodes() (nodes []*cache.Node, err error) {
	defer Return(&err)

	nodes = g.cache.AllNodes()
	if len(nodes) == 0 {
		// cache expired
		nodes, err := g.getNodes("")
		Ck(err)
	}

	for _, node := range nodes {
		// add to cache
		_, found := g.cache.byname[node.Name]
		if found {
			// XXX handle
			log.Printf("duplicate filename: %s", node.Name)
		}
		g.cache.byname[node.Name] = node
	}

	return
}

func (g *Google) Getnode(fn string) (node *cache.Node, err error) {
	defer Return(&err)

	// refresh cache
	_, err = g.AllNodes()
	Ck(err)

	// return nil if not found
	node, _ = g.cache.byname[fn]
	// Pf("%#v\n", g.cache)

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

func (g *Google) NewNode(f *drive.File) (node *cache.Node, err error) {
	defer Return(&err)
	// XXX s/mcp/g.docPrefix/g
	re := regexp.MustCompile(`^mcp-(\d+)-`)
	m := re.FindStringSubmatch(f.Title)
	var num int
	if len(m) == 2 {
		num, _ = strconv.Atoi(m[1])
	}

	node = &Node{
		File:    f,
		Name:    f.Title,
		Num:     num,
		Created: f.CreatedDate,
		URL:     f.AlternateLink,
	}

	err = c.Addnode(node)
	Ck(err)
	return
}
