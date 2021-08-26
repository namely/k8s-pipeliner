# assign the current version from the binary
VERSION = $(shell go run cmd/k8s-pipeliner/main.go --version | awk '{print $$3}')

install:
	go install -mod=vendor ./...

test:
	go test ./...
	make lint

.PHONY: deps
deps:
	export GO111MODULE="on"
	go mod vendor
	go get github.com/mattn/goveralls
	go get github.com/go-playground/overalls
	go install github.com/mgechev/revive@latest

.PHONY: coveralls
coveralls:
	overalls -project=github.com/namely/k8s-pipeliner -covermode=count
	goveralls -coverprofile=overalls.coverprofile -service=travis-ci

.PHONY: lint
lint:
	revive ./pipeline/...
