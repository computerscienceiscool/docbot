package google

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"

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
	MinNextNum int
	fnre       *regexp.Regexp
}

// NewFolder returns an object that represents a single gdrive folder.
// We assume that the folder is accessible by the service account
// json credentials provided in cbuf.
func NewFolder(cbuf []byte, folderid string, docPattern *regexp.Regexp, minNextNum int) (gf *Folder, err error) {
	defer Return(&err)

	gf = &Folder{id: folderid, MinNextNum: minNextNum}

	ctx := context.Background()

	gf.docs, err = docs.NewService(ctx, option.WithCredentialsJSON(cbuf))
	Ck(err)

	gf.drive, err = drive.NewService(ctx, option.WithCredentialsJSON(cbuf))
	Ck(err)

	gf.fnre = docPattern

	return
}

func (gf *Folder) Doc2json(node *Node) (buf []byte, err error) {
	defer Return(&err)
	doc, err := gf.docs.Documents.Get(node.Id()).Do()
	Ck(err)
	b := doc.Body
	buf, err = json.MarshalIndent(b.Content, "", "  ")
	Ck(err)
	return
}

func (gf *Folder) Doc2txt(node *Node) (txt string, err error) {
	defer Return(&err)
	// https://github.com/rsbh/doc2md/blob/a740060638ca55813c25c7e4a6cf7774e3cbd63f/pkg/transformer/doc2json.go#L368
	// XXX fetch doc in mkNode
	// XXX move node stuff to Node, include gf in struct
	doc, err := gf.docs.Documents.Get(node.Id()).Do()
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

func (gf *Folder) textRuns(node *Node) (els []*docs.ParagraphElement, err error) {
	defer Return(&err)
	doc, err := gf.docs.Documents.Get(node.Id()).Do()
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

func (gf *Folder) FindTextRun(node *Node, txt string) (el *docs.ParagraphElement, err error) {
	defer Return(&err)

	els, err := gf.textRuns(node)
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

func (gf *Folder) GetHeaders(node *Node) (h map[string]string, err error) {
	defer Return(&err)
	h = make(map[string]string)
	txt, err := gf.Doc2txt(node)
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

func (gf *Folder) QueryNodes(query string) (nodes []*Node, err error) {
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
		Ck(err, query)

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

func (gf *Folder) Rm(rmnode *Node) (err error) {
	defer Return(&err)
	if rmnode == nil {
		return
	}
	err = gf.drive.Files.Delete(rmnode.id).Do()
	Ck(err)
	return
}

func (gf *Folder) Copy(tnode *Node, newName string) (node *Node, err error) {
	defer Return(&err)
	parentref := &drive.ParentReference{Id: gf.id}
	file := &drive.File{Parents: []*drive.ParentReference{parentref}, Title: newName}
	f, err := gf.drive.Files.Copy(tnode.Id(), file).Do()
	Ck(err)
	node = gf.mkNode(f)
	return
}

/*

func (gf *Folder) put() {

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
