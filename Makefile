VERSION ?= $(shell git describe --tags --always --dirty)
LDFLAGS := -ldflags "-X main.version=${VERSION}"

.PHONY: build
build:
	go build ${LDFLAGS} -o bin/rprompt cmd/rprompt/main.go

.PHONY: install
install: build
	cp bin/rprompt /usr/local/bin/rprompt

.PHONY: clean
clean:
	rm -rf bin/

.PHONY: test
test:
	go test ./... 