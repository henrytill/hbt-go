.SUFFIXES:

GO = go

GOPATH = $(shell $(GO) env GOPATH)
STATICCHECK = $(GOPATH)/bin/staticcheck

GO_SOURCES =
GO_SOURCES += cmd/hbt/main.go
GO_SOURCES += internal/types.go
GO_SOURCES += internal/parser.go
GO_SOURCES += internal/formatter.go
GO_SOURCES += internal/parser/markdown.go
GO_SOURCES += internal/parser/xml.go
GO_SOURCES += internal/parser/pinboard.go
GO_SOURCES += internal/parser/html.go
GO_SOURCES += internal/formatter/html.go
GO_SOURCES += internal/formatter/yaml.go

BIN =
BIN += hbt
BIN += testgen

BIN_TARGETS = $(addprefix bin/,$(BIN))

.PHONY: all
all: $(BIN_TARGETS)

bin/testgen:: cmd/testgen/main.go
	$(GO) build -o $@ $<

bin/%:: $(GO_SOURCES)
	$(GO) build -o $@ cmd/$*/main.go

.PHONY: lint
lint:
	$(GO) vet ./...
	$(STATICCHECK) ./...

.PHONY: fmt
fmt:
	$(GO) fmt ./...

hbt_test.go: bin/testgen
	$<

.PHONY: test
test: bin/hbt hbt_test.go internal/test_support.go
	$(GO) test -v

.PHONY: clean
clean:
	rm -f $(BIN_TARGETS) hbt_test.go
