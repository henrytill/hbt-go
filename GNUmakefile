.SUFFIXES:

GO = go
CTAGS = ctags
ETAGS = etags

STATICCHECK = staticcheck
DEADCODE = deadcode

BIN =
BIN += hbt
BIN += pinboard

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
SOURCES += internal/pinboard/note.go
SOURCES += internal/pinboard/types.go
SOURCES += internal/parser/pinboard/json.go
SOURCES += internal/parser/pinboard/xml.go
SOURCES += internal/client/pinboard/client.go
SOURCES += internal/client/pinboard/credentials.go
SOURCES += internal/client/pinboard/notes.go
SOURCES += internal/client/pinboard/posts.go
SOURCES += internal/client/pinboard/tags.go

BIN_TARGETS = $(addprefix $(BINDIR)/,$(BIN))

all: $(BIN_TARGETS)

$(BINDIR):
	mkdir -p $@

$(BINDIR)/%: export CGO_ENABLED = 0
$(BINDIR)/%: cmd/%/*.go $(SOURCES) | $(BINDIR)
	$(GO) build -o $@ ./cmd/$*/

lint:
	$(GO) vet ./...
	$(STATICCHECK) ./...
	$(DEADCODE) -test ./...

fmt:
	$(GO) fmt ./...

tags:
	@printf '%s\n' $(SOURCES) | $(CTAGS) -L -

TAGS:
	@printf '%s\n' $(SOURCES) | $(ETAGS) -L -

test: $(BIN_TARGETS)
	$(GO) generate ./test
	$(GO) test -v ./...

clean:
	rm -f $(BIN_TARGETS)

distclean: clean
	rmdir $(BINDIR)

.PHONY: all lint fmt tags TAGS test clean distclean
