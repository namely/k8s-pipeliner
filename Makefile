# assign the current version from the binary
VERSION = $(shell go run cmd/k8s-pipeliner/main.go --version | awk '{print $$3}')

install:
	go install -mod=vendor ./...

test:
	GO111MODULE=on go test ./...
	GO111MODULE=on go list ./... | grep -v /vendor/ | xargs -L1 golint -set_exit_status

.PHONY: deps
deps:
	GO111MODULE=on go mod vendor
	go get github.com/mattn/goveralls
	go get github.com/go-playground/overalls

.PHONY: coveralls
coveralls:
	overalls -project=github.com/namely/k8s-pipeliner -covermode=count
	goveralls -coverprofile=overalls.coverprofile -service=travis-ci

.PHONY: lint
lint:
	golangci-lint run --skip-dirs=vendor --skip-dirs=gen --skip-dirs=mocks --deadline=5m --tests=true -E golint \
    -E gosec -E unconvert -E goconst -E gocyclo -E goimports
