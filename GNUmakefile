.SUFFIXES:

GO = go

GOPATH = $(shell $(GO) env GOPATH)
STATICCHECK = $(GOPATH)/bin/staticcheck

GO_SOURCES =
GO_SOURCES += cmd/hbt/main.go
GO_SOURCES += internal/types.go
GO_SOURCES += internal/parser.go
GO_SOURCES += internal/formatter.go
GO_SOURCES += internal/mappings.go
GO_SOURCES += internal/parser/markdown.go
GO_SOURCES += internal/parser/xml.go
GO_SOURCES += internal/parser/pinboard.go
GO_SOURCES += internal/parser/html.go
GO_SOURCES += internal/formatter/html.go
GO_SOURCES += internal/formatter/yaml.go

BIN =
BIN += hbt

BIN_TARGETS = $(addprefix bin/,$(BIN))

.PHONY: all
all: $(BIN_TARGETS)

bin/%: $(GO_SOURCES)
	$(GO) build -o $@ cmd/$*/main.go

.PHONY: lint
lint:
	$(GO) vet ./...
	$(STATICCHECK) ./...

.PHONY: fmt
fmt:
	$(GO) fmt ./...

.PHONY: test
test: bin/hbt
	$(GO) generate ./integration
	$(GO) test -v ./integration

.PHONY: clean
clean:
	rm -f $(BIN_TARGETS)
