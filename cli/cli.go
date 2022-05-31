package cli

import (
	"embed"
	"os"
	"text/template"

	"github.com/stevegt/docbot/bot"
	. "github.com/stevegt/goadapt"
)

//go:embed template/*
var fs embed.FS

func Run(b *bot.Bot) (err error) {
	defer Return(&err)

	err = b.Init()
	Ck(err)

	var tname string
	switch true {
	case b.Ls:
		tname = "ls.txt"
	default:
		Assert(false, "unhandled: %#v", b)
	}

	t, err := template.ParseFS(fs, "template/*")
	Ck(err)

	tx := b.StartTransaction()
	defer tx.Close()

	err = t.ExecuteTemplate(os.Stdout, tname, tx)
	Ck(err)

	return
}

/*
func ls(b *bot.Bot) (out []byte, err error) {
	defer Return(&err)

	tmpl, err := template.New("ls").Parse()

	nodes, err := tx.AllNodes()
	Ck(err)
	for _, n := range nodes {
		out = append(out, []byte(Spf("%s (%s) (%s)\n", n.Name(), n.Id(), n.MimeType()))...)
	}
	return
}
*/
