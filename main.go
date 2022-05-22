package main

import (
	"os"

	"github.com/docopt/docopt-go"
	"github.com/stevegt/docbot/bot"
	. "github.com/stevegt/goadapt"
)

const usage = `docbot

Usage:
  docbot ls 
  docbot put 
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
	res, err := b.Run()
	Ck(err)
	Pl(res)
}
