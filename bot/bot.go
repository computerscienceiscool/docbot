package bot

import (
	"bytes"
	"context"
	"embed"
	"fmt"
	"html/template"
	"io/ioutil"
	"time"

	. "github.com/stevegt/goadapt"
	"google.golang.org/api/docs/v1"
	"google.golang.org/api/drive/v2"
	"google.golang.org/api/option"
)

type Bot struct {
	Ls       bool
	Put      bool
	Serve    bool
	Credpath string
	Folderid string
	docs     *docs.Service
	drive    *drive.Service
}

func (b *Bot) Init() (err error) {
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
		res = b.ls()
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
}

func (b *Bot) indexHtml() (out []byte) {
	files := b.getFiles()
	t := template.Must(template.ParseFS(fs, "template/index.html"))
	var buf bytes.Buffer
	err := t.Execute(&buf, files)
	Ck(err)
	out = buf.Bytes()
	return
}

func (b *Bot) ls() (out []byte) {
	for _, f := range b.getFiles() {
		out = append(out, []byte(Spf("%s (%s) (%s)\n", f.Title, f.Id, f.MimeType))...)
	}
	return
}

func (b *Bot) getFiles() (files []*drive.File) {

	query := fmt.Sprintf("'%v' in parents", b.Folderid)

	var pageToken string
	for {

		q := b.drive.Files.List().Q(query)

		if pageToken != "" {
			q = q.PageToken(pageToken)
		}

		res, err := q.Do()
		Ck(err)

		for _, f := range res.Items {
			// f is a drive.File
			files = append(files, f)
		}

		pageToken = res.NextPageToken
		if pageToken == "" {
			break
		}
	}

	// Pl("items", i)
	return
}
