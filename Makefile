NAME := botsrv

GOFLAGS=-mod=vendor

PKG := `go list ${GOFLAGS} -f {{.Dir}} ./...`

ifeq ($(RACE),1)
	GOFLAGS+=-race
endif

LINT_VERSION := v1.50.1

MAIN := cmd/${NAME}/main.go

VERSION?=$(git version > /dev/null 2>&1 && git describe --dirty=-dirty --always 2>/dev/null || echo NO_VERSION)
LDFLAGS=-ldflags "-X=main.version=$(VERSION)"

tools:
	@go install github.com/vmkteam/mfd-generator@latest
	@go install github.com/vmkteam/zenrpc/v2/zenrpc@latest
	@curl -sfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin ${LINT_VERSION}

fmt:
	@goimports -local ${NAME} -l -w $(PKG)

lint:
	@golangci-lint run -c .golangci.yml

build:
	@CGO_ENABLED=0 go build $(LDFLAGS) $(GOFLAGS) -o ${NAME} $(MAIN)

run:
	@echo "Compiling"
	@go run $(LDFLAGS) $(GOFLAGS) $(MAIN) -config=cfg/local.toml -verbose -verbose-sql

generate:
	@go generate ./pkg/rpc

test:
	@echo "Running tests"
	@go test -count=1 $(LDFLAGS) $(GOFLAGS) -coverprofile=coverage.txt -covermode count $(PKG)

test-short:
	@go test $(LDFLAGS) $(GOFLAGS) -v -test.short -test.run="Test[^D][^B]" -coverprofile=coverage.txt -covermode count $(PKG)

mod:
	@go mod tidy
	@go mod vendor
	@git add vendor

NS := "NONE"

MAPPING := "common:users;vfs:vfsFiles,vfsFolders"

mfd-xml:
	@mfd-generator xml -c "postgres://postgres:postgres@localhost:5432/botsrv?sslmode=disable" -m ./docs/model/botsrv.mfd -n $(MAPPING)
mfd-model:
	@mfd-generator model -m ./docs/model/botsrv.mfd -p db -o ./pkg/db
mfd-repo: --check-ns
	@mfd-generator repo -m ./docs/model/botsrv.mfd -p db -o ./pkg/db -n $(NS)
mfd-xml-lang:
	#TODO: add namespaces support for xml-lang command
	@mfd-generator xml-lang  -m ./docs/model/botsrv.mfd


--check-ns:
ifeq ($(NS),"NONE")
	$(error "You need to set NS variable before run this command. For example: NS=common make $(MAKECMDGOALS) or: make $(MAKECMDGOALS) NS=common")
endif
