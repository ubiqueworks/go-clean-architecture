.PHONY: build clean clean-all deps gofmt gazelle package protos test update

BAZEL:=$(shell which bazel)
DEP:=$(shell which dep)
GAZELLE:=$(shell which gazelle)
GOFMT:=$(shell which gofmt)
GOIMPORTS:=$(shell which goimports)
PROTOC:=$(shell which protoc)

PROTO_SOURCES:=$(shell find . -type f -name '*.proto')
PROTO_FILES:=$(patsubst %.proto,%.pb.go,$(PROTO_SOURCES))

all: package

clean:
	@$(BAZEL) clean

clean-all:
	@$(BAZEL) clean --expunge

deps:
	@$(DEP) ensure

gofmt:
	@$(GOFMT) -w -s framework/
	@$(GOFMT) -w -s service/

goimports:
	@$(GOIMPORTS) -w framework/
	@$(GOIMPORTS) -w service/

gazelle:
	@$(BAZEL) run //:gazelle

protos: $(PROTO_FILES)

$(PROTO_FILES): %.pb.go: %.proto

%.pb.go:
	@echo Compiling $<
	@$(PROTOC) --go_out=plugins=grpc:. $<

package: update
	@$(BAZEL) run --experimental_platforms=@io_bazel_rules_go//go/toolchain:linux_amd64 //:package

build: gofmt gazelle
	@bazel build //...

test: gazelle
	@bazel test //...

update: goimports gazelle protos
