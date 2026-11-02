# Copyright (c) 2026 J
# All rights reserved.
# Released under the GNU Affero General Public License v3.0.

# audit2fivew – Makefile
# Builds a fully static Go binary (CGO_ENABLED=0) and optional RPM.

NAME     := audit2fivew
VERSION  ?= 1.0.0
COMMIT   := $(shell git rev-parse --short HEAD 2>/dev/null)

GOOS     ?= $(shell go env GOOS)
GOARCH   ?= $(shell go env GOARCH)

GOFLAGS  := -trimpath

ifeq ($(COMMIT),)
LDFLAGS  := -s -w -X main.Version=$(VERSION)
else
LDFLAGS  := -s -w -X main.Version=$(VERSION)-$(COMMIT)
endif

DOCS_DIR ?= public

# Pure-Go build — no CGO, no external C libraries.
export CGO_ENABLED := 0
export GOOS
export GOARCH

.PHONY: build build-linux build-linux-amd64 build-linux-arm64 clean test rpm vendor docs docs-serve

## build: compile a static binary for current OS/arch → ./audit2fivew
build:
	go build $(GOFLAGS) -ldflags '$(LDFLAGS)' -o $(NAME) .

## build-linux: cross-compile static ELF binary for Linux x86_64 → ./audit2fivew
build-linux: build-linux-amd64

## build-linux-amd64: cross-compile static ELF binary for Linux x86_64
build-linux-amd64:
	GOOS=linux GOARCH=amd64 go build $(GOFLAGS) -ldflags '$(LDFLAGS)' -o $(NAME) .

## build-linux-arm64: cross-compile static ELF binary for Linux aarch64
build-linux-arm64:
	GOOS=linux GOARCH=arm64 go build $(GOFLAGS) -ldflags '$(LDFLAGS)' -o $(NAME) .

## test: run the full test suite
test:
	go test ./...

## docs: generate static HTML documentation from doc comments using doc2go
docs:
	@which doc2go >/dev/null 2>&1 || (echo "doc2go not found. Install with: go install go.abhg.dev/doc2go@latest" && exit 1)
	@mkdir -p $(DOCS_DIR)
	doc2go -out $(DOCS_DIR) ./...
	@echo "Documentation built in $(DOCS_DIR)/"

## docs-serve: build and serve documentation locally at http://localhost:8080
docs-serve: docs
	@echo "Serving documentation at http://localhost:8080 (Ctrl+C to stop)..."
	python3 -m http.server 8080 -d $(DOCS_DIR)

## clean: remove build artefacts
clean:
	rm -f $(NAME) $(NAME)-linux-*
	rm -rf rpmbuild vendor $(DOCS_DIR)

## vendor: vendor Go dependencies (needed for offline / RPM builds)
vendor:
	go mod vendor

## rpm: build an RPM suitable for RHEL / EPEL
rpm: vendor
	mkdir -p rpmbuild/{BUILD,RPMS,SOURCES,SPECS,SRPMS}
	tar czf rpmbuild/SOURCES/$(NAME)-$(VERSION).tar.gz \
		--transform 's,^,$(NAME)-$(VERSION)/,' \
		--exclude=rpmbuild --exclude=.git --exclude=vendor \
		main.go doc.go go.mod go.sum \
		internal/ \
		Makefile $(NAME).spec README.md LICENSE $(NAME).1
	tar czf rpmbuild/SOURCES/$(NAME)-$(VERSION)-vendor.tar.gz vendor/
	rpmbuild -ba \
		--define "_topdir $(CURDIR)/rpmbuild" \
		$(NAME).spec
	@echo "RPM(s) written to rpmbuild/RPMS/ and rpmbuild/SRPMS/"

