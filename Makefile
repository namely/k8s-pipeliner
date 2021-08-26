# assign the current version from the binary
VERSION = $(shell go run cmd/k8s-pipeliner/main.go --version | awk '{print $$3}')

install:
	go install -mod=vendor ./...

test:
	GO111MODULE="on" go test ./...
	GO111MODULE="on" make lint

.PHONY: deps
deps:
	go mod vendor
	go get github.com/mattn/goveralls
	go get github.com/go-playground/overalls
	go get github.com/mgechev/revive
	GO111MODULE="on" go get github.com/stretchr/testify@v1.4.0

.PHONY: coveralls
coveralls:
	overalls -project=github.com/namely/k8s-pipeliner -covermode=count
	goveralls -coverprofile=overalls.coverprofile -service=travis-ci

.PHONY: lint
lint:
	revive ./pipeline/...
