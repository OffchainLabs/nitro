#
# Copyright 2020, Offchain Labs, Inc. All rights reserved.
#

precompile_names = AddressTable Aggregator BLS FunctionTable GasInfo Info osTest Owner RetryableTx Statistics Sys
precompiles = $(patsubstr %,./solgen/generated/%.go, $(precompile_names))

color_pink = "\e[38;5;161;1m"
color_reset = "\e[0;0m"

done = "%bdone!%b\n" $(color_pink) $(color_reset)


# user targets

.make/all: always .make/solgen .make/solidity .make/test
	@printf "%bdone building %s%b\n" $(color_pink) $$(expr $$(echo $? | wc -w) - 1) $(color_reset)
	@touch .make/all

contracts: .make/solidity
	@printf $(done)

format fmt: .make/fmt
	@printf $(done)

lint: .make/lint
	@printf $(done)

test: .make/test
	gotestsum --format short-verbose
	@printf $(done)

push: .make/push
	@printf "%bready for push!%b\n" $(color_pink) $(color_reset)

clean:
	go clean -testcache		
	@rm -rf solgen/artifacts solgen/cache solgen/go/
	@rm -f .make/*


# regular build rules



# strategic rules to minimize dependency building

.make/push: .make/lint
	make $(MAKEFLAGS) .make/test
	@touch .make/push

.make/lint: .golangci.yml *.go */*.go */*/*.go .make/solgen
	golangci-lint run --fix
	@touch .make/lint

.make/fmt: .golangci.yml *.go */*.go */*/*.go .make/solgen
	golangci-lint run --disable-all -E gofmt --fix
	@touch .make/fmt

.make/test: *.go */*.go */*/*.go .make/solgen .make/solidity
	cd arbos && gotestsum --format short-verbose
	@touch .make/test

.make/solgen: solgen/gen.go .make/solidity
	mkdir -p solgen/go/
	go run solgen/gen.go
	@touch .make/solgen

.make/solidity: solgen/src/*.sol | .make
	yarn --cwd solgen build
	@touch .make/solidity

.make:
	yarn --cwd solgen install
	mkdir .make


# Makefile settings

always:              # use this to force other rules to always build
.DELETE_ON_ERROR:    # causes a failure to delete its target
