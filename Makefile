# assign the current version from the binary
VERSION = $(shell go run cmd/k8s-pipeliner/main.go --version | awk '{print $$3}')

test:
	go test -cover ./...
	golint -set_exit_status ./...

build:
	mkdir -p bin/darwin
	mkdir -p bin/linux
	GOOS=darwin go build -o bin/darwin/k8s-pipeliner cmd/k8s-pipeliner/main.go
	GOOS=linux go build -o bin/linux/k8s-pipeliner cmd/k8s-pipeliner/main.go

release: test build;
	git tag v$(VERSION) && git push --tags
	github-release release \
    --user namely \
    --repo k8s-pipeliner \
    --tag v$(VERSION) \
    --name "k8s-pipeliner release $(VERSION)" \
    --description "";
	github-release upload \
		--user namely \
		--repo k8s-pipeliner \
		--tag v$(VERSION) \
		--name "k8s-pipeliner-osx-amd64" \
		--file bin/darwin/k8s-pipeliner;
	github-release upload \
		--user namely \
		--repo k8s-pipeliner \
		--tag v$(VERSION) \
		--name "k8s-pipeliner-linux-amd64" \
		--file bin/linux/k8s-pipeliner;
