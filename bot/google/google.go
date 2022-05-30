package google

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
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

var headre = regexp.MustCompile(`^(\w+):\s+(.*)`)

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
	id         string
	docs       *docs.Service
	drive      *drive.Service
	minNextNum int
	fnre       *regexp.Regexp
	mu         sync.Mutex
	// txcache *transaction
}

// NewFolder returns an object that represents a single gdrive folder.
// We assume that the folder is accessible by the service account
// json credentials provided in cbuf.
func NewFolder(cbuf []byte, folderid, docPrefix string, minNextNum int) (gf *Folder, err error) {
	defer Return(&err)

	gf = &Folder{id: folderid, minNextNum: minNextNum}

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

func (gf *Folder) StartTransaction() (tx *transaction) {
	gf.mu.Lock()
	tx = &transaction{gf: gf}
	tx.nodes = []*Node{}
	tx.byname = make(map[string]*Node)

	// XXX use gf.txcache to store a prestaged tx populated with nodes
	/*
		if time.Now().Sub(tx.time) > time.Minute {
			gf.clearcache()
		}
		tx.time = time.Now()
	*/

	return
}

type transaction struct {
	gf      *Folder
	nodes   []*Node
	byname  map[string]*Node
	lastNum int
	start   time.Time
	loaded  bool
}

func (tx *transaction) loadNodes() (err error) {
	defer Return(&err)
	_, err = tx.AllNodes()
	Ck(err)
	return
}

func (tx *transaction) Close() {
	defer tx.gf.mu.Unlock()
}

// AllNodes returns all nodes and caches the results.
func (tx *transaction) AllNodes() (nodes []*Node, err error) {
	defer Return(&err)

	if !tx.loaded {
		// populate node list
		nodes, err := tx.queryNodes("")
		Ck(err)
		for _, node := range nodes {
			err = tx.cachenode(node)
			Ck(err)
		}
		tx.loaded = true
	}

	// nodes = make([]*Node, len(tx.nodes))
	// copy(nodes, tx.nodes)
	return tx.nodes, nil
}

func (tx *transaction) Copy(fileId, newName string) (node *Node, err error) {
	defer Return(&err)
	parentref := &drive.ParentReference{Id: tx.gf.id}
	file := &drive.File{Parents: []*drive.ParentReference{parentref}, Title: newName}
	f, err := tx.gf.drive.Files.Copy(fileId, file).Do()
	Ck(err)
	node = tx.gf.mkNode(f)
	err = tx.cachenode(node)
	Ck(err)
	return
}

func (tx *transaction) Doc2json(node *Node) (buf []byte, err error) {
	defer Return(&err)
	doc, err := tx.gf.docs.Documents.Get(node.Id()).Do()
	Ck(err)
	b := doc.Body
	buf, err = json.MarshalIndent(b.Content, "", "  ")
	Ck(err)
	return
}

func (tx *transaction) Doc2txt(node *Node) (txt string, err error) {
	defer Return(&err)
	// https://github.com/rsbh/doc2md/blob/a740060638ca55813c25c7e4a6cf7774e3cbd63f/pkg/transformer/doc2json.go#L368
	// XXX fetch doc in mkNode
	// XXX move node stuff to Node, include gf in struct
	doc, err := tx.gf.docs.Documents.Get(node.Id()).Do()
	Ck(err)
	b := doc.Body
	// Pprint(b.Content)
	// iterate over elements
	for _, s := range b.Content {
		if s.Paragraph != nil {
			for _, el := range s.Paragraph.Elements {
				if el.TextRun != nil {
					content := el.TextRun.Content
					txt += content
				}
			}
		}
	}
	// replace line tabulation unicode chars with newline
	txt = strings.ReplaceAll(txt, "\u000b", "\n")
	return
}

// FindNodes takes a query string and returns all matching nodes.
func (tx *transaction) FindNodes(query string) (nodes []*Node, err error) {
	defer Return(&err)
	nodes, err = tx.queryNodes(query)
	Ck(err)
	return
}

func (tx *transaction) textRuns(node *Node) (els []*docs.ParagraphElement, err error) {
	defer Return(&err)
	doc, err := tx.gf.docs.Documents.Get(node.Id()).Do()
	Ck(err)
	b := doc.Body
	// iterate over elements
	for _, s := range b.Content {
		if s.Paragraph != nil {
			for _, el := range s.Paragraph.Elements {
				if el.TextRun != nil {
					els = append(els, el)
				}
			}
		}
	}
	return
}

func (tx *transaction) FindTextRun(node *Node, txt string) (el *docs.ParagraphElement, err error) {
	defer Return(&err)

	els, err := tx.textRuns(node)
	Ck(err)
	for _, e := range els {
		// Pprint(e.TextRun.Content)
		if e.TextRun.Content == txt {
			el = e
			break
		}
	}

	return
}

func (tx *transaction) Getnode(fn string) (node *Node, err error) {
	defer Return(&err)
	err = tx.loadNodes()
	Ck(err)
	// return nil if not found
	node, _ = tx.byname[fn]
	return
}

// return the last used document number
func (tx *transaction) LastNum() (last int, err error) {
	defer Return(&err)
	err = tx.loadNodes()
	Ck(err)
	last = tx.lastNum
	return
}

// return the next (unused) document number
func (tx *transaction) NextNum() (next int, err error) {
	defer Return(&err)
	err = tx.loadNodes()
	Ck(err)
	last, err := tx.LastNum()
	Ck(err)
	next = last + 1
	if next < tx.gf.minNextNum {
		next = tx.gf.minNextNum
	}
	return
}

func (tx *transaction) GetHeaders(node *Node) (h map[string]string, err error) {
	defer Return(&err)
	h = make(map[string]string)
	txt, err := tx.Doc2txt(node)
	Ck(err)
	lines := strings.Split(txt, "\n")
	for _, line := range lines {
		if line == "" {
			break
		}
		m := headre.FindStringSubmatch(line)
		if m == nil {
			// XXX handle?
			Pl("unmatched header line:", line)
			continue
		}
		Assert(len(m) == 3)
		h[m[1]] = m[2]
	}
	return
}

func (tx *transaction) cachenode(node *Node) (err error) {
	defer Return(&err)
	_, found := tx.byname[node.name]
	if found {
		// XXX figure out how to handle this better -- this could be
		// caused either by a race condition, by a user forcing a filename
		// in the URL, or by an error in code; likely need a better
		// better warning path to user and to dev
		// XXX for now we might just add an integer suffix
		log.Printf("duplicate filename: %s", node.name)
	}
	tx.byname[node.name] = node
	tx.nodes = append(tx.nodes, node)
	if node.num > tx.lastNum {
		tx.lastNum = node.num
	}
	return
}

// XXX move to backend
func (tx *transaction) queryNodes(query string) (nodes []*Node, err error) {
	defer Return(&err)

	if query == "" {
		query = fmt.Sprintf("'%v' in parents", tx.gf.id)
	} else {
		query = fmt.Sprintf("'%v' in parents and %s", tx.gf.id, query)
	}

	var pageToken string
	for {

		q := tx.gf.drive.Files.List().Q(query)

		if pageToken != "" {
			q = q.PageToken(pageToken)
		}

		res, err := q.Do()
		Ck(err)

		for _, f := range res.Items {
			// f is a *drive.File
			node := tx.gf.mkNode(f)
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

func (tx *transaction) XXXReplaceText(node *Node, parms map[string]string) (res *docs.BatchUpdateDocumentResponse, err error) {
	defer Return(&err)
	reqs := replaceAllTextRequest(parms)
	res, err = tx.batchUpdate(node, reqs)
	return
}

func replaceAllTextRequest(parms map[string]string) (reqs []*docs.Request) {
	reqs = make([]*docs.Request, 0)
	for k, v := range parms {
		reqs = append(reqs, &docs.Request{
			ReplaceAllText: &docs.ReplaceAllTextRequest{
				ContainsText: &docs.SubstringMatchCriteria{
					MatchCase: true,
					Text:      k,
				},
				ReplaceText: v,
			},
		})
	}
	return
}

func (tx *transaction) batchUpdate(node *Node, reqs []*docs.Request) (res *docs.BatchUpdateDocumentResponse, err error) {
	update := &docs.BatchUpdateDocumentRequest{Requests: reqs}
	res, err = tx.gf.docs.Documents.BatchUpdate(node.id, update).Do()
	Ck(err)
	return
}

func (tx *transaction) Rm(fn string) (err error) {
	defer Return(&err)
	rmnode, err := tx.Getnode(fn)
	Ck(err)
	if rmnode == nil {
		return
	}
	err = tx.gf.drive.Files.Delete(rmnode.id).Do()
	Ck(err)
	var newNodes []*Node
	for _, n := range tx.nodes {
		if n.id != rmnode.id {
			newNodes = append(newNodes, n)
		}
	}
	tx.nodes = newNodes
	delete(tx.byname, fn)
	return
}

func updateLinkRequest(el *docs.ParagraphElement, url string) (req *docs.Request) {
	req = &docs.Request{
		UpdateTextStyle: &docs.UpdateTextStyleRequest{
			Fields: "link",
			Range: &docs.Range{
				StartIndex: el.StartIndex,
				EndIndex:   el.EndIndex,
			},
			TextStyle: &docs.TextStyle{
				Link: &docs.Link{
					Url: url,
				},
			},
		},
	}
	return
}

// open or create file
// XXX pass in opts struct instead of http.Request
func (tx *transaction) Opendoc(r *http.Request, template, filename, baseUrl string) (node *Node, err error) {
	defer Return(&err)
	node, err = tx.Getnode(filename)
	Ck(err)
	if node == nil {
		// file doesn't exist -- create it
		node, err = tx.mkdoc(r, template, filename, baseUrl)
		Ck(err)
		Assert(node != nil, "%s, %s, %s", template, filename, r.Form.Get("title"))
	}
	return
}

// create file
func (tx *transaction) mkdoc(r *http.Request, template, filename, baseUrl string) (node *Node, err error) {
	defer Return(&err)
	// get template
	Assert(len(template) > 0)
	tnode, err := tx.Getnode(template)
	Ck(err, template)
	Assert(tnode != nil, template)

	node, err = tx.Copy(tnode.Id(), filename)
	Ck(err)

	title := r.Form.Get("title")
	date := r.Form.Get("session_date")
	speakers := r.Form.Get("session_speakers")

	v := url.Values{}
	v.Set("filename", node.name)
	v.Set("unlock", "t")
	unlockUrl := Spf("%s?%s", baseUrl, v.Encode())

	if len(title) == 0 {
		// XXX handle
		log.Printf("missing title: %s, %v", r.URL, r.Form)
	}

	// generate update requests
	parms := map[string]string{
		"NAME":             filename,
		"TITLE":            title,
		"SESSION_DATE":     date,
		"SESSION_SPEAKERS": speakers,
		"UNLOCK_URL":       unlockUrl,
	}
	treqs := replaceAllTextRequest(parms)
	el, err := tx.FindTextRun(node, "UNLOCK_URL")
	Ck(err)
	var reqs []*docs.Request
	if el != nil {
		lreq := updateLinkRequest(el, unlockUrl)
		reqs = append(treqs, lreq)
	} else {
		reqs = treqs
	}
	res, err := tx.batchUpdate(node, reqs)
	Ck(err)

	// Pprint(res)
	_ = res

	return
}

/*
	// find link
	el, err := tx.FindTextRun(node, "UNLOCK_URL")
	Tassert(t, err == nil, err)
	Tassert(t, el != nil, Spf("%#v", node))
	Tassert(t, el.TextRun.Content == "UNLOCK_URL", el)

	// generate update requests
	lreq := updateLinkRequest(el, unlockUrl)
	parms := map[string]string{"UNLOCK_URL": unlockUrl}
	treqs := replaceAllTextRequest(parms)
	reqs := append(treqs, lreq)

	// run it
	res, err := tx.batchUpdate(node, reqs)
	Tassert(t, err == nil, err)
	// Pprint(res)
	_ = res
*/

/*
func (tx *transaction) ReplaceWithUrl(node *Node, tag, text, url string) (err error) {
	defer Return(&err)

	// https://github.com/rsbh/doc2md/blob/a740060638ca55813c25c7e4a6cf7774e3cbd63f/pkg/transformer/doc2json_test.go#L132
	newtr := &docs.TextRun{
		Content: text,
		TextStyle: &docs.TextStyle{
			Link: &docs.Link{
				Url: url,
			},
		},
	}

	// XXX call textRuns()
	doc, err := tx.gf.docs.Documents.Get(node.Id()).Do()
	Ck(err)
	b := doc.Body
	// iterate over elements
	for _, s := range b.Content {
		if s.Paragraph != nil {
			for _, el := range s.Paragraph.Elements {
				if el.TextRun != nil {
					if el.TextRun.Content == tag {
						// replace with new link
						el.TextRun = newtr
					}
				}
			}
		}
	}
	return
}
*/

/*

func (gf *Google) Copy(fileid, newName string) (newNode *cache.Node, err error) {
	defer Return(&err)
	req := request{name: fn}
	res := gf.xeq(tx.GetNode, req)
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

/*
	// https://github.com/rsbh/doc2md/blob/a740060638ca55813c25c7e4a6cf7774e3cbd63f/pkg/transformer/doc2json.go#L368
	// XXX fetch doc in mkNode
	// XXX move node stuff to Node, include gf in struct
	doc, err := gf.docs.Documents.Get(node.Id()).Do()
	Ck(err)
	b := doc.Body
	// iterate over elements
loop:
	for _, s := range b.Content {
		if s.Paragraph != nil {
			for _, el := range s.Paragraph.Elements {
				if el.TextRun != nil {
					content := el.TextRun.Content
					if content == "\n" {
						break loop
					}
					m := headre.FindStringSubmatch(content)
					if m == nil {
						// XXX handle?
						Pl("unmatched header line:", content)
						Pprint(s)
						continue
					}
					Assert(len(m) == 3)
					h[m[1]] = m[2]
				}
			}
		}
	}
	return
}
*/
