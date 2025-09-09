.SUFFIXES:

GO = go

GOPATH = $(shell $(GO) env GOPATH)
STATICCHECK = $(GOPATH)/bin/staticcheck
DEADCODE = $(GOPATH)/bin/deadcode

BIN =
BIN += hbt

BINDIR = bin

SOURCES =
SOURCES += internal/formats.go
SOURCES += internal/types/types.go
SOURCES += internal/mappings.go
SOURCES += internal/formatter/types.go
SOURCES += internal/formatter/html.go
SOURCES += internal/formatter/yaml.go
SOURCES += internal/parser/types.go
SOURCES += internal/parser/html.go
SOURCES += internal/parser/markdown.go
SOURCES += internal/pinboard/types.go
SOURCES += internal/parser/pinboard/json.go
SOURCES += internal/parser/pinboard/xml.go

BIN_TARGETS = $(addprefix $(BINDIR)/,$(BIN))

all: $(BIN_TARGETS)

$(BINDIR):
	mkdir -p $@

$(BINDIR)/%: cmd/%/main.go $(SOURCES) | $(BINDIR)
	$(GO) build -o $@ $<

lint:
	$(GO) vet ./...
	$(STATICCHECK) ./...
	$(DEADCODE) -test ./...

fmt:
	$(GO) fmt ./...

test: $(BIN_TARGETS)
	$(GO) generate ./test
	$(GO) test -v ./test

clean:
	rm -f $(BIN_TARGETS)

distclean: clean
	rmdir $(BINDIR)

.PHONY: all lint fmt test clean distclean
