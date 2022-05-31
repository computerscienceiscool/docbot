package google

import (
	. "github.com/stevegt/goadapt"
	"google.golang.org/api/docs/v1"
)

type batch struct {
	gf   *Folder
	reqs []*docs.Request
}

func (gf *Folder) BatchStart() (b *batch) {
	b = &batch{gf: gf}
	// b.reqs = make([]*docs.Request, 0)
	return
}

func (b *batch) ReplaceAllTextRequest(parms map[string]string) {
	for k, v := range parms {
		b.reqs = append(b.reqs, &docs.Request{
			ReplaceAllText: &docs.ReplaceAllTextRequest{
				ContainsText: &docs.SubstringMatchCriteria{
					MatchCase: true,
					Text:      k,
				},
				ReplaceText: v,
			},
		})
	}
}

func (b *batch) UpdateLinkRequest(el *docs.ParagraphElement, url string) {
	req := &docs.Request{
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
	b.reqs = append(b.reqs, req)
	return
}

func (b *batch) Run(node *Node) (res *docs.BatchUpdateDocumentResponse, err error) {
	update := &docs.BatchUpdateDocumentRequest{Requests: b.reqs}
	res, err = b.gf.docs.Documents.BatchUpdate(node.id, update).Do()
	Ck(err)
	return
}
