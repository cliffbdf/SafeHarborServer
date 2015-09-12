# Makefile for building the Safe Harbor Server.
# This does not deploy any servers: it merely complies and packages the code.


PRODUCTNAME=Safe Harbor Server
ORG=Scaled Markets
VERSION=1.0
BUILD=1234
EXECNAME=SafeHarborServer

.DELETE_ON_ERROR:
.ONESHELL:
.SUFFIXES:
.DEFAULT_GOAL: all

SHELL = /bin/sh

CURDIR=$(shell pwd)

#GO_LDFLAGS=-ldflags "-X `go list ./version`.Version $(VERSION)"

.PHONY: all compile authcert vm deploy clean info
.DEFAULT: all

src_dir = $(CURDIR)/src

build_dir = $(CURDIR)/../bin

GOPATH = $(CURDIR)

all: compile authcert

$(build_dir):
	mkdir $(build_dir)

$(build_dir)/$(EXECNAME): $(build_dir) $(src_dir)/main

# 'make compile' builds the executable, which is placed in <build_dir>.
compile: $(build_dir)/$(EXECNAME)
	@echo GOPATH=$(GOPATH)
	go build -o $(build_dir)/$(EXECNAME) main

clean:
	rm -r -f $(build_dir)
	rm -r -f $(test_build_dir)

info:
	@echo "Makefile for $(PRODUCTNAME)"

