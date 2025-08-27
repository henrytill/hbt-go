.SUFFIXES:

GO = go

GOPATH = $(shell $(GO) env GOPATH)
STATICCHECK = $(GOPATH)/bin/staticcheck

BIN =
BIN += hbt

BIN_TARGETS = $(addprefix bin/,$(BIN))

all: $(BIN_TARGETS)

bin/%:
	$(GO) build -o $@ cmd/$*/main.go

lint:
	$(GO) vet ./...
	$(STATICCHECK) ./...

fmt:
	$(GO) fmt ./...

test: $(BIN_TARGETS)
	$(GO) generate ./integration
	$(GO) test -v ./integration

clean:
	rm -f $(BIN_TARGETS)

.PHONY: all lint fmt test clean
