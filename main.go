package main

import (
	"os"

	"github.com/docopt/docopt-go"
	"github.com/stevegt/docbot/bot"
	"github.com/stevegt/docbot/cli"
	"github.com/stevegt/docbot/web"
	"github.com/stevegt/envi"
	. "github.com/stevegt/goadapt"
)

const usage = `docbot

Usage:
  docbot ls 
  docbot serve 

  If DOCBOT_CONF is not set to a config file path, then docbot will look
  for a file named ".docbot.conf" in the local directory.

`

func main() {
	parser := &docopt.Parser{OptionsFirst: false}
	o, err := parser.ParseArgs(usage, os.Args[1:], "0.0")
	Ck(err)
	var b bot.Bot
	err = o.Bind(&b)
	Ck(err)
	b.Confpath = envi.String("DOCBOT_CONF", ".docbot.conf")
	b.Credpath = envi.String("DOCBOT_CRED", ".docbot.cred")

	if b.Serve {
		err = web.Serve(&b)
	} else {
		err = cli.Run(&b)
	}
	if err != nil {
		Fpf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

}
