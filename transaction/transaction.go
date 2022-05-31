package transaction

import (
	"log"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/stevegt/docbot/google"
	. "github.com/stevegt/goadapt"
)

type Transaction struct {
	gf      *google.Folder
	nodes   []*google.Node
	byname  map[string]*google.Node
	lastNum int
	start   time.Time
	loaded  bool
}

var mu sync.Mutex

func Start(gf *google.Folder) (tx *Transaction) {
	mu.Lock()
	tx = &Transaction{gf: gf}
	tx.nodes = []*google.Node{}
	tx.byname = make(map[string]*google.Node)

	// XXX use gf.txcache to store a prestaged tx populated with nodes
	/*
		if time.Now().Sub(tx.time) > time.Minute {
			gf.clearcache()
		}
		tx.time = time.Now()
	*/

	return
}

func (tx *Transaction) Close() {
	tx.gf = nil
	mu.Unlock()
}

func (tx *Transaction) loadNodes() (err error) {
	defer Return(&err)
	_, err = tx.AllNodes()
	Ck(err)
	return
}

// AllNodes returns all nodes and caches the results.
func (tx *Transaction) AllNodes() (nodes []*google.Node, err error) {
	defer Return(&err)

	if !tx.loaded {
		// populate node list
		nodes, err := tx.gf.QueryNodes("")
		Ck(err)
		for _, node := range nodes {
			err = tx.cachenode(node)
			Ck(err)
		}
		tx.loaded = true
	}

	// nodes = make([]*google.Node, len(tx.nodes))
	// copy(nodes, tx.nodes)
	return tx.nodes, nil
}

// FindNodes takes a query string and returns all matching nodes.
func (tx *Transaction) FindNodes(query string) (nodes []*google.Node, err error) {
	defer Return(&err)
	nodes, err = tx.gf.QueryNodes(query)
	Ck(err)
	return
}

func (tx *Transaction) Getnode(fn string) (node *google.Node, err error) {
	defer Return(&err)
	err = tx.loadNodes()
	Ck(err)
	// return nil if not found
	node, _ = tx.byname[fn]
	return
}

// return the last used document number
func (tx *Transaction) LastNum() (last int, err error) {
	defer Return(&err)
	err = tx.loadNodes()
	Ck(err)
	last = tx.lastNum
	return
}

// return the next (unused) document number
func (tx *Transaction) NextNum() (next int, err error) {
	defer Return(&err)
	err = tx.loadNodes()
	Ck(err)
	last, err := tx.LastNum()
	Ck(err)
	next = last + 1
	if next < tx.gf.MinNextNum {
		next = tx.gf.MinNextNum
	}
	return
}

func (tx *Transaction) cachenode(node *google.Node) (err error) {
	defer Return(&err)
	_, found := tx.byname[node.Name()]
	if found {
		// XXX figure out how to handle this better -- this could be
		// caused either by a race condition, by a user forcing a filename
		// in the URL, or by an error in code; likely need a better
		// better warning path to user and to dev
		// XXX for now we might just add an integer suffix
		log.Printf("duplicate filename: %s", node.Name())
	}
	tx.byname[node.Name()] = node
	tx.nodes = append(tx.nodes, node)
	if node.Num() > tx.lastNum {
		tx.lastNum = node.Num()
	}
	return
}

// open or create file
// XXX pass in opts struct instead of http.Request
func (tx *Transaction) Opendoc(r *http.Request, template, filename, baseUrl string) (node *google.Node, err error) {
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
func (tx *Transaction) mkdoc(r *http.Request, template, filename, baseUrl string) (node *google.Node, err error) {
	defer Return(&err)
	// get template
	Assert(len(template) > 0)
	tnode, err := tx.Getnode(template)
	Ck(err, template)
	Assert(tnode != nil, template)

	node, err = tx.Copy(tnode, filename)
	Ck(err)

	title := r.Form.Get("title")
	date := r.Form.Get("session_date")
	speakers := r.Form.Get("session_speakers")

	v := url.Values{}
	v.Set("filename", node.Name())
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
	batch := tx.gf.BatchStart()
	batch.ReplaceAllTextRequest(parms)
	el, err := tx.gf.FindTextRun(node, "UNLOCK_URL")
	Ck(err)
	if el != nil {
		batch.UpdateLinkRequest(el, unlockUrl)
	}
	res, err := batch.Run(node)
	Ck(err)

	// Pprint(res)
	_ = res

	return
}

func (tx *Transaction) Rm(rmnode *google.Node) (err error) {
	defer Return(&err)
	if rmnode == nil {
		return
	}
	err = tx.gf.Rm(rmnode)
	Ck(err)
	var newNodes []*google.Node
	for _, n := range tx.nodes {
		if n.Id() != rmnode.Id() {
			newNodes = append(newNodes, n)
		}
	}
	tx.nodes = newNodes
	delete(tx.byname, rmnode.Name())
	return
}

func (tx *Transaction) Copy(tnode *google.Node, newName string) (node *google.Node, err error) {
	defer Return(&err)
	node, err = tx.gf.Copy(tnode, newName)
	Ck(err)
	err = tx.cachenode(node)
	Ck(err)
	return
}
