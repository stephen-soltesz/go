
LINESERVER=../../../../../../bin/lineviewer
all: resources/clientview.js resources.go $(LINESERVER)

resources/clientview.js: gopherjs/clientview.go
	gopherjs build -o ./resources/clientview.js gopherjs/clientview.go

resources.go: resources/*
	go-bindata -o=./resources.go ./resources
	gofmt -w ./resources.go

$(LINESERVER): *.go
	go install github.com/stephen-soltesz/go/cmds/lineviewer
