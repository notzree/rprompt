VERSION ?= $(shell git describe --tags --always --dirty)
LDFLAGS := -ldflags "-X main.version=${VERSION}"
RELEASE_DIR := bin/release
MAIN_FILE := ./main.go

.PHONY: build
build:
	go build ${LDFLAGS} -o bin/rprompt ${MAIN_FILE}

.PHONY: install
install: build
	cp bin/rprompt /usr/local/bin/rprompt

.PHONY: clean
clean:
	rm -rf bin/

.PHONY: test
test:
	go test ./... 

.PHONY: release-dirs
release-dirs:
	mkdir -p ${RELEASE_DIR}

.PHONY: release-darwin-arm64
release-darwin-arm64: release-dirs
	GOOS=darwin GOARCH=arm64 go build ${LDFLAGS} -o ${RELEASE_DIR}/rprompt ${MAIN_FILE}
	cd ${RELEASE_DIR} && tar -czf rprompt-darwin-arm64.tar.gz rprompt
	rm ${RELEASE_DIR}/rprompt

.PHONY: release-darwin-amd64
release-darwin-amd64: release-dirs
	GOOS=darwin GOARCH=amd64 go build ${LDFLAGS} -o ${RELEASE_DIR}/rprompt ${MAIN_FILE}
	cd ${RELEASE_DIR} && tar -czf rprompt-darwin-amd64.tar.gz rprompt
	rm ${RELEASE_DIR}/rprompt

.PHONY: release
release: clean release-darwin-arm64 release-darwin-amd64
	@echo "Release artifacts created in ${RELEASE_DIR}/"
	@echo "\nSHA256 checksums:"
	@cd ${RELEASE_DIR} && shasum -a 256 *.tar.gz 


test:
	go test ./... -v




