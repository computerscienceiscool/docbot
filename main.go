package main

import (
	"os"

	"github.com/docopt/docopt-go"
	"github.com/stevegt/docbot/bot"
	. "github.com/stevegt/goadapt"
)

const usage = `docbot

Usage:
  docbot <credpath> <folderid> ls 
  docbot <credpath> <folderid> put 
  docbot <credpath> <folderid> serve 

`

func main() {
	parser := &docopt.Parser{OptionsFirst: false}
	o, _ := parser.ParseArgs(usage, os.Args[1:], "0.0")
	var b bot.Bot
	err := o.Bind(&b)
	Ck(err)

	res, err := b.Run()
	Ck(err)
	Pl(res)
}
