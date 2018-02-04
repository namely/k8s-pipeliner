# assign the current version from the binary
VERSION = $(shell go run cmd/k8s-pipeliner/main.go --version | awk '{print $$3}')

install:
	go install ./...
	
test:
	go test ./...
	golint -set_exit_status ./...

.PHONY: deps
deps:
	go get -u github.com/kardianos/govendor
	govendor sync
	go get github.com/mattn/goveralls
	go get github.com/go-playground/overalls

.PHONY: coveralls
coveralls:
	overalls -project=github.com/namely/k8s-pipeliner -covermode=count
	goveralls -coverprofile=overalls.coverprofile -service=travis-ci