#!/bin/bash

# used in development

pidfn=/run/user/$UID/docbot

killpid() {
	# kill $(cat $pidfn)
	killall docbot
}

trap "killpid; exit" SIGINT SIGTERM EXIT

winid=$(getwinid)
base=$PWD

export DOCBOT_CONF=bot/testdata/docbot.conf
export DOCBOT_CRED=local/mcpbot-mcpbot-key.json

set -x
while true
do
	padsp signalgen -t 100m sin 523 # C
	cd $base
	inotifywait -r -e modify *
	padsp signalgen -t 100m sin 262 # C
	killpid
	sleep 1
	go vet ./... || continue
	wmctrl -ia $winid
	padsp signalgen -t 100m sin 330 # E 
	cd bot
	go test -v -coverprofile=/tmp/got.out -coverpkg=./... || continue
	padsp signalgen -t 100m sin 392 # G
	goenv exec go tool cover -html=/tmp/got.out -o /tmp/got.html
	xdg-open /tmp/got.html
	sleep 1
	cd $base
	go run . serve &
	sleep 2
	if ! kill -0 $(cat $pidfn) 
	then
		padsp signalgen -t 100m sin 262
		wmctrl -ia $winid
		continue
	fi
	xdg-open http://localhost:8080
	sleep 1
done
