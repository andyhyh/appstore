GO_EXECUTABLE ?= go
PACKAGE_DIRS := $(shell glide nv)
BINDIR := $(CURDIR)/bin

.PHONY: build
build:
	GOBIN=$(BINDIR) ${GO_EXECUTABLE} install github.com/uninett/appstore/cmd/...

.PHONY: bootstrap
bootstrap:
	glide install --strip-vendor
	scripts/setup-apimachinery.sh
	mkdir -p $(BINDIR)
