package cache

import (
	"log"
	"sync"
	"time"

	. "github.com/stevegt/goadapt"
	"google.golang.org/api/drive/v2"
)

type Node struct {
	File     *drive.File
	Name     string
	Id       string
	URL      string
	MimeType string
	Num      int
	Created  string
}

type Cache struct {
	nodes   []*Node
	byname  map[string]*Node
	time    time.Time
	lastNum int
	mu      sync.Mutex
}

func NewCache() (c *Cache) {
	c = &Cache{}
	c.clear()
	return
}

func (c *Cache) clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.nodes = []*Node{}
	c.byname = make(map[string]*Node)
}

func (c *Cache) Addnode(node *Node) (err error) {
	defer Return(&err)
	c.mu.Lock()
	defer c.mu.Unlock()
	_, found := c.byname[node.Name]
	if found {
		// XXX figure out how to handle this better -- this could be
		// caused either by a race condition, by a user forcing a filename
		// in the URL, or by an error in code; likely need a better
		// better warning path to user and to dev
		// XXX for now we might just add an integer suffix
		log.Printf("duplicate filename: %s", node.Name)
	}
	c.byname[node.Name] = node
	c.nodes = append(c.nodes, node)
	if node.Num > c.lastNum {
		c.lastNum = node.Num
	}
	return
}

func (c *Cache) AllNodes() (nodes []*Node, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if time.Now().Sub(c.time) > time.Minute {
		return
	}
	return c.nodes
}
	g.cache.time = time.Now()
