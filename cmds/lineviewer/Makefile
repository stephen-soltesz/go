
all: resources/clientview.js resources.go server

resources/clientview.js: gopherjs/clientview.go
	gopherjs build -o ./resources/clientview.js gopherjs/clientview.go

resources.go: resources/*
	go-bindata -o=./resources.go ./resources

server: *.go
	go build server*.go resources.go
