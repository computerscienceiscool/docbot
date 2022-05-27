package google

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"sync"
	"time"

	. "github.com/stevegt/goadapt"
	"google.golang.org/api/docs/v1"
	"google.golang.org/api/drive/v2"
	"google.golang.org/api/option"
)

/*

architecture:

- caller calls NewFolder() goroutine for each folder accessed
	- create folder{}
	- return folder{}

- folder exports methods e.g. Copy(), AllNodes, NewNode, LastNum
	- when any of these is called:
		- lock
		- call non-exported backend method

- backend methods
	- maintain internal state, refreshing or modifying
	  cache as needed

*/

type Node struct {
	file     *drive.File
	name     string
	id       string
	url      string
	mimeType string
	num      int
	created  string
}

func (n *Node) Name() string     { return n.name }
func (n *Node) Id() string       { return n.id }
func (n *Node) URL() string      { return n.url }
func (n *Node) MimeType() string { return n.mimeType }
func (n *Node) Num() int         { return n.num }
func (n *Node) Created() string  { return n.created }

func (gf *Folder) mkNode(f *drive.File) (node *Node) {
	m := gf.fnre.FindStringSubmatch(f.Title)
	var num int
	if len(m) == 2 {
		num, _ = strconv.Atoi(m[1])
	}

	node = &Node{
		file:     f,
		name:     f.Title,
		id:       f.Id,
		url:      f.AlternateLink,
		mimeType: f.MimeType,
		num:      num,
		created:  f.CreatedDate,
	}

	return
}

type Folder struct {
	id    string
	docs  *docs.Service
	drive *drive.Service
	c     *cache
	mu    sync.Mutex
	fnre  *regexp.Regexp
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
func NewFolder(cbuf []byte, folderid, docPrefix string) (gf *Folder, err error) {
	defer Return(&err)

	gf = &Folder{id: folderid}
	gf.clearcache()

	ctx := context.Background()

	gf.docs, err = docs.NewService(ctx, option.WithCredentialsJSON(cbuf))
	Ck(err)

	gf.drive, err = drive.NewService(ctx, option.WithCredentialsJSON(cbuf))
	Ck(err)

	pat := Spf("^%s-(\\d+)-", docPrefix)
	gf.fnre, err = regexp.Compile(pat)
	Ck(err)

	return
}

// AllNodes returns all nodes and caches the results.
func (gf *Folder) AllNodes() (nodes []*Node, err error) {
	gf.mu.Lock()
	defer gf.mu.Unlock()
	defer Return(&err)

	nodes, err = gf.allNodes()
	Ck(err)
	return
}

func (gf *Folder) allNodes() (nodes []*Node, err error) {
	defer Return(&err)

	if time.Now().Sub(gf.c.time) > time.Minute {
		gf.clearcache()
	}
	gf.c.time = time.Now()

	if len(gf.c.nodes) == 0 {
		// cache expired
		nodes, err := gf.queryNodes("")
		Ck(err)
		for _, node := range nodes {
			err = gf.cachenode(node)
			Ck(err)
		}
	}

	nodes = make([]*Node, len(gf.c.nodes))
	copy(nodes, gf.c.nodes)
	return
}

func (gf *Folder) Copy(fileId, newName string) (node *Node, err error) {
	gf.mu.Lock()
	defer gf.mu.Unlock()
	defer Return(&err)

	node, err = gf.cp(fileId, newName)
	Ck(err)
	return
}

func (gf *Folder) cp(fileId, newName string) (node *Node, err error) {
	defer Return(&err)
	parentref := &drive.ParentReference{Id: gf.id}
	file := &drive.File{Parents: []*drive.ParentReference{parentref}, Title: newName}
	f, err := gf.drive.Files.Copy(fileId, file).Do()
	Ck(err)
	node = gf.mkNode(f)
	err = gf.cachenode(node)
	Ck(err)
	return
}

// FindNodes takes a query string and returns all matching nodes.
func (gf *Folder) FindNodes(query string) (nodes []*Node, err error) {
	gf.mu.Lock()
	defer gf.mu.Unlock()
	defer Return(&err)

	nodes, err = gf.findNodes(query)
	Ck(err)
	return
}

func (gf *Folder) findNodes(query string) (nodes []*Node, err error) {
	defer Return(&err)
	nodes, err = gf.queryNodes(query)
	Ck(err)
	return
}

func (gf *Folder) Getnode(fn string) (node *Node, err error) {
	gf.mu.Lock()
	defer gf.mu.Unlock()
	defer Return(&err)

	node, err = gf.getnode(fn)
	Ck(err)
	return
}

func (gf *Folder) getnode(fn string) (node *Node, err error) {
	defer Return(&err)
	// refresh cache
	_, err = gf.allNodes()
	Ck(err)
	// return nil if not found
	node, _ = gf.c.byname[fn]
	return
}

// return the last used document number
func (gf *Folder) LastNum() (last int, err error) {
	gf.mu.Lock()
	defer gf.mu.Unlock()
	defer Return(&err)

	last, err = gf.lastNum()
	Ck(err)
	return
}

func (gf *Folder) lastNum() (last int, err error) {
	defer Return(&err)
	// refresh cache
	_, err = gf.allNodes()
	Ck(err)
	last = gf.c.lastNum
	return
}

func (gf *Folder) cachenode(node *Node) (err error) {
	defer Return(&err)
	_, found := gf.c.byname[node.name]
	if found {
		// XXX figure out how to handle this better -- this could be
		// caused either by a race condition, by a user forcing a filename
		// in the URL, or by an error in code; likely need a better
		// better warning path to user and to dev
		// XXX for now we might just add an integer suffix
		log.Printf("duplicate filename: %s", node.name)
	}
	gf.c.byname[node.name] = node
	gf.c.nodes = append(gf.c.nodes, node)
	if node.num > gf.c.lastNum {
		gf.c.lastNum = node.num
	}
	return
}

func (gf *Folder) Clearcache() {
	gf.mu.Lock()
	defer gf.mu.Unlock()
	gf.clearcache()
}

func (gf *Folder) clearcache() {
	gf.c = &cache{}
	gf.c.nodes = []*Node{}
	gf.c.byname = make(map[string]*Node)
}

func (gf *Folder) queryNodes(query string) (nodes []*Node, err error) {
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
			node := gf.mkNode(f)
			Ck(err)
			nodes = append(nodes, node)
		}

		pageToken = res.NextPageToken
		if pageToken == "" {
			break
		}
	}

	return
}

func (gf *Folder) Rm(fn string) (err error) {
	gf.mu.Lock()
	defer gf.mu.Unlock()
	defer Return(&err)

	err = gf.rm(fn)
	Ck(err)
	return
}

func (gf *Folder) rm(fn string) (err error) {
	defer Return(&err)
	node, err := gf.getnode(fn)
	Ck(err)
	if node == nil {
		return
	}
	err = gf.drive.Files.Delete(node.id).Do()
	Ck(err)
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
/*
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

*/

/*
type response struct {
	node    Node
	nodes   []Node
	lastNum int
	err     error
}

func (res response) Error() string {
	return res.err.Error()
}

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
*/
