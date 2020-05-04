# set default shell
SHELL = bash -e -o pipefail

# Variables
VERSION                  ?= $(shell cat ./VERSION)

default: test

help:
	@echo "Usage: make [<target>]"
	@echo "where available targets are:"
	@echo
	@echo "help              : Print this help"
	@echo "test              : Run unit tests, if any"
	@echo "test              : Run unit tests, if any"
	@echo "protoc 			 : Build protos"

test:
	go test -race -v -p 1 ./...

protoc:
	docker build -t local/protogen -f protos/Dockerfile.protogen .
	docker run --name protogen local/protogen
	docker cp protogen:/home/protos/go ./protos
	docker rm protogen
