all:
	go fmt ./...
	go vet ./...
	golint ./...

install:
	go install ./ember-serve
