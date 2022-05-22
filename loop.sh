#!/bin/bash

# used in development

trap "kill $pid; exit" SIGINT SIGTERM EXIT

winid=$(getwinid)
base=$PWD

set -x
while true
do
	padsp signalgen -v -t 100m sin 523 # C
	cd $base
	inotifywait -r -e modify *
	padsp signalgen -v -t 100m sin 262 # C
	kill $(cat /run/user/$UID/docbot)
	sleep 1
	go vet ./... || continue
	padsp signalgen -v -t 100m sin 330 # E 
	cd bot
	if ! go test -v 
	then
		wmctrl -ia $winid
		continue
	fi
	padsp signalgen -v -t 100m sin 392 # G
	cd $base
	sleep 1
	go run . local/mcpbot-mcpbot-key.json 1HcCIw7ppJZPD9GEHccnkgNYUwhAGCif6 serve &
	pid=$!
	sleep 1
	xdg-open http://localhost:8080
done
