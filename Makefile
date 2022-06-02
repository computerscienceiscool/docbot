image = docbot
tag = 0.0
# imagefq = us.gcr.io/fleet-cirrus-766/$(image):$(tag)
imagefq = stevegt/$(image):$(tag)
host = mcp.systems

# token = $(shell gcloud auth application-default print-access-token)

all:

login:
	# docker login -u oauth2accesstoken -p $(token) https://us.gcr.io
	docker login 

build: 
	go vet ./...
	# go test -v ./...
	go build
	docker build -t $(imagefq) .

push: build
	# test `git status --porcelain | wc -l` -eq 0
	docker push $(imagefq)

run: 
	ssh $(host) docker pull $(imagefq)
	ssh $(host) docker run -v docbot-data:/data --restart unless-stopped -d --network=host --name=$(image) $(imagefq)

kill:
	- ssh $(host) docker kill $(image)

clean: stop
	ssh $(host) docker container prune -f 
	ssh $(host) docker image prune -f
	ssh $(host) docker container ps -a

stop:
	- ssh $(host) docker stop $(image)

rm:
	- ssh $(host) docker rm $(image)

start:
	ssh $(host) docker start $(image)

restart: stop rm run

bash:
	ssh -t $(host) docker exec -it $(image) bash

tail:
	ssh -t $(host) docker exec -it $(image) supervisorctl tail -f docbot

logs:
	ssh -t $(host) docker logs $(image)
