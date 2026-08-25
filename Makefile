# Build outputs go to ./bin, which is gitignored.
#
# The reason this file exists: `go build ./tools/tenant` writes ./tenant into the
# repo root, with no extension for a glob to catch, and a later `git add -A`
# commits 60MB of Mach-O. That happened. Every target here passes -o so the
# output lands somewhere ignored.
BIN := bin
GO  ?= go

# GOROOT is inherited stale in this environment — it points at a mise install
# directory that no longer exists, which makes every `go` invocation fail with
# "cannot find GOROOT directory" even though `go` itself is on PATH and fine.
# The toolchain resolves its own root correctly when the variable is simply
# absent, so remove it for every recipe rather than trying to guess the right
# value here.
unexport GOROOT

.PHONY: build tools apiary agent clean test lint

## build: everything, into ./bin
build: apiary agent tools

## apiary: the control plane
apiary:
	$(GO) build -o $(BIN)/apiary .

## agent: the host agent (its own module)
agent:
	cd agent && GOWORK=off $(GO) build -o ../$(BIN)/apiary-agent .

## tools: the Atlas external_schema programs
##
## atlas.hcl invokes these with `go run ./tools/...` rather than a built binary,
## so these targets are for running them by hand. Building them in the root is
## what produced the committed binaries.
tools:
	$(GO) build -o $(BIN)/schema-control ./tools/control
	$(GO) build -o $(BIN)/schema-tenant ./tools/tenant

## test: the whole suite. TEST_DATABASE_URL must point at a database whose name
## contains "test" — testsupport refuses anything else, because this suite
## CREATEs and DROPs schemas.
test:
	$(GO) test ./... -count=1

lint:
	gofmt -l . | grep -v '^ui/' | grep -v '^sdk/' && exit 1 || true
	$(GO) vet ./...

clean:
	rm -rf $(BIN)
