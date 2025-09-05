.SUFFIXES:

GO = go

GOPATH = $(shell $(GO) env GOPATH)
STATICCHECK = $(GOPATH)/bin/staticcheck

BIN =
BIN += hbt

SOURCES =
SOURCES += cmd/hbt/main.go
SOURCES += internal/formatter.go
SOURCES += internal/formatter/html.go
SOURCES += internal/formatter/yaml.go
SOURCES += internal/mappings.go
SOURCES += internal/parser.go
SOURCES += internal/parser/html.go
SOURCES += internal/parser/markdown.go
SOURCES += internal/parser/pinboard.go
SOURCES += internal/parser/xml.go
SOURCES += internal/types.go

BIN_TARGETS = $(addprefix bin/,$(BIN))

all: $(BIN_TARGETS)

bin/%: $(SOURCES)
	$(GO) build -o $@ cmd/$*/main.go

lint:
	$(GO) vet ./...
	$(STATICCHECK) ./...

fmt:
	$(GO) fmt ./...

test: $(BIN_TARGETS)
	$(GO) generate ./test
	$(GO) test -v ./test

clean:
	rm -f $(BIN_TARGETS)

.PHONY: all lint fmt test clean
