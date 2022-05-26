package google

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"time"

	"github.com/stevegt/goadapt"
	. "github.com/stevegt/goadapt"
	"google.golang.org/api/docs/v1"
	"google.golang.org/api/drive/v2"
	"google.golang.org/api/option"
)

/*

architecture:

- caller calls NewFolder() goroutine for each folder accessed
	- create internal request channel
	- create folder{}
	- return folder{}

- folder exports methods e.g. Copy(), AllNodes, NewNode, LastNum
	- when any of these is called:
		- make response channel
		- generate request
			- includes response channel
		- send to folder.run() goroutine via internal request channel
		- listen for response
		- return response to caller

- folder.run() goroutine
	- listen on request channel
	- send responses by value, not by reference
	- maintain its own internal state, refreshing or modifying its gdrive
	  cache as needed

*/

type Node struct {
	file     *drive.File
	Name     string
	Id       string
	URL      string
	MimeType string
	Num      int
	Created  string
}

type Folder struct {
	id       string
	docs     *docs.Service
	drive    *drive.Service
	requests chan request
	c        *cache
}

type cache struct {
	nodes   []*Node
	byname  map[string]*Node
	time    time.Time
	lastNum int
}

// NewFolder returns an object that represents a single gdrive folder.
// We assume that the folder is accessible by the service account
// json credentials provided in cbuf.
func NewFolder(cbuf []byte, folderid string) (gf *Folder, err error) {
	defer Return(&err)

	gf = &Folder{id: folderid}

	ctx := context.Background()

	gf.docs, err = docs.NewService(ctx, option.WithCredentialsJSON(cbuf))
	Ck(err)

	gf.drive, err = drive.NewService(ctx, option.WithCredentialsJSON(cbuf))
	Ck(err)

	gf.requests = gf.run()

	return
}

func (gf *Folder) xeq(op op, req request) (res response) {
	req.folder = gf

	// attach op func
	req.op = op

	// make response channel
	req.responses = make(chan response)

	// send to folder.run() goroutine via internal request channel
	gf.requests <- req

	// listen for response
	res = <-req.responses

	return
}

type op func(request) response

type request struct {
	op        op
	folder    *Folder
	fileId    string
	name      string
	newName   string
	query     string
	responses chan response
}

/*
func ckchan(responses chan response, err error) {
	if err != nil {
		panic(response{err: err})
	}
}
*/

// convert panic into returned err on channel
func returnRes(res *response) {
	r := recover()
	if r == nil {
		return
	}
	switch concrete := r.(type) {
	// case *response:
	// responses <- *concrete
	case *goadapt.AdaptErr:
		res = &response{err: concrete}
	default:
		// wasn't us -- re-raise
		panic(r)
	}
}

func (gf *Folder) cp(req request) (res response) {
	defer returnRes(&res)
	parentref := &drive.ParentReference{Id: gf.id}
	file := &drive.File{Parents: []*drive.ParentReference{parentref}, Title: req.newName}
	f, err := gf.drive.Files.Copy(req.fileId, file).Do()
	Ck(err)
	newNode, err := gf.newNode(f)
	Ck(err)
	res = response{node: *newNode}
	return
}

func (gf *Folder) newNode(f *drive.File) (node *Node, err error) {
	defer Return(&err)
	// XXX s/mcp/g.docPrefix/g
	re := regexp.MustCompile(`^mcp-(\d+)-`)
	m := re.FindStringSubmatch(f.Title)
	var num int
	if len(m) == 2 {
		num, _ = strconv.Atoi(m[1])
	}

	node = &Node{
		file:    f,
		Name:    f.Title,
		Num:     num,
		Created: f.CreatedDate,
		URL:     f.AlternateLink,
	}

	Ck(err)
	return
}

func (gf *Folder) addnode(node *Node) (err error) {
	defer Return(&err)
	_, found := gf.c.byname[node.Name]
	if found {
		// XXX figure out how to handle this better -- this could be
		// caused either by a race condition, by a user forcing a filename
		// in the URL, or by an error in code; likely need a better
		// better warning path to user and to dev
		// XXX for now we might just add an integer suffix
		log.Printf("duplicate filename: %s", node.Name)
	}
	gf.c.byname[node.Name] = node
	gf.c.nodes = append(gf.c.nodes, node)
	if node.Num > gf.c.lastNum {
		gf.c.lastNum = node.Num
	}
	return
}

type response struct {
	node    Node
	nodes   []Node
	lastNum int
	err     error
}

/*
func (res response) Error() string {
	return res.err.Error()
}
*/

func (gf *Folder) run() (requests chan request) {
	gf.clearcache()

	// create internal request channel
	requests = make(chan request)

	go func() {
		// serve callers
		for req := range requests {
			req.responses <- req.op(req)
		}
	}()

	return
}

func (gf *Folder) Copy(fileId, newName string) (newNode *Node, err error) {
	defer Return(&err)
	req := request{fileId: fileId, newName: newName}
	res := gf.xeq(gf.cp, req)
	return &res.node, err
}

// AllNodes returns all nodes and caches the results.
func (gf *Folder) AllNodes() (nodes []Node, err error) {
	defer Return(&err)
	req := request{}
	res := gf.xeq(gf.allNodes, req)
	Pl("lkajflsakjfd", res)
	return res.nodes, err
}

func (gf *Folder) allNodes(req request) (res response) {
	defer returnRes(&res)

	if time.Now().Sub(gf.c.time) > time.Minute {
		gf.clearcache()
	}

	if len(gf.c.nodes) == 0 {
		// cache expired
		nodes, err := gf.getNodes("")
		Ck(err)
		for _, node := range nodes {
			err = gf.addnode(node)
			Ck(err)
		}
		gf.c.time = time.Now()
	}

	res = response{}
	for _, node := range gf.c.nodes {
		res.nodes = append(res.nodes, *node)
	}

	return
}

func (gf *Folder) clearcache() {
	gf.c = &cache{}
	gf.c.nodes = []*Node{}
	gf.c.byname = make(map[string]*Node)
}

func (gf *Folder) getNodes(query string) (nodes []*Node, err error) {
	defer Return(&err)

	if query == "" {
		query = fmt.Sprintf("'%v' in parents", gf.id)
	} else {
		query = fmt.Sprintf("'%v' in parents and %s", gf.id, query)
	}

	var pageToken string
	for {

		q := gf.drive.Files.List().Q(query)

		if pageToken != "" {
			q = q.PageToken(pageToken)
		}

		res, err := q.Do()
		Ck(err)

		for _, f := range res.Items {
			// f is a *drive.File
			_, err := gf.newNode(f)
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
func (gf *Folder) FindNodes(query string) (nodes []Node, err error) {
	defer Return(&err)
	req := request{query: query}
	res := gf.xeq(gf.findNodes, req)
	return res.nodes, err
}

func (gf *Folder) findNodes(req request) (res response) {
	defer returnRes(&res)
	nodes, err := gf.getNodes(req.query)
	Ck(err)
	res = response{}
	for _, node := range nodes {
		res.nodes = append(res.nodes, *node)
	}
	return
}

// return the last used document number
func (gf *Folder) LastNum() (last int, err error) {
	defer Return(&err)
	req := request{}
	res := gf.xeq(gf.lastNum, req)
	return res.lastNum, err
}

func (gf *Folder) lastNum(req request) (res response) {
	defer returnRes(&res)
	// refresh cache
	res = gf.allNodes(req)
	last := gf.c.lastNum
	res.lastNum = last
	return
}

func (gf *Folder) Getnode(fn string) (node *Node, err error) {
	defer Return(&err)
	req := request{name: fn}
	res := gf.xeq(gf.getNode, req)
	return &res.node, err
}

func (gf *Folder) getNode(req request) (res response) {
	defer returnRes(&res)
	// refresh cache
	res = gf.allNodes(req)
	// return empty node if not found
	node, _ := gf.c.byname[req.name]
	res.node = *node
	return
}

/*

func (gf *Google) Copy(fileid, newName string) (newNode *cache.Node, err error) {
	defer Return(&err)
	req := request{name: fn}
	res := gf.xeq(gf.getNode, req)
	return &res.node, err
}




	parentref := &drive.ParentReference{Id: g.folderid}
	file := &drive.File{Parents: []*drive.ParentReference{parentref}, Title: newName}
	f, err := g.drive.Files.Copy(fileid, file).Do()
	Ck(err)
	newNode, err = g.NewNode(f)
	Ck(err)
	return
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

func (gf *Google) put() {

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

*/
