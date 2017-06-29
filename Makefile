GO_EXECUTABLE ?= go
PACKAGE_DIRS := $(shell glide nv)
BINDIR := $(CURDIR)/bin

.PHONY: build
build:
	GOBIN=$(BINDIR) ${GO_EXECUTABLE} install github.com/UNINETT/appstore/cmd/appstore-server

.PHONY: deps
deps:
	glide install --strip-vendor
	scripts/setup-apimachinery.sh
	mkdir -p $(BINDIR)

.PHONY: test
test:
	go test github.com/UNINETT/appstore/cmd/...
