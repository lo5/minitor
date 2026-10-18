BIN     := minitor
FLAGS   := -trimpath -ldflags="-s -w"
INSTDIR := $(or $(shell go env GOBIN),$(shell go env GOPATH)/bin)

.PHONY: all build install uninstall test clean

all: build

build:
	go build $(FLAGS) -o $(BIN) .

install:
	go install $(FLAGS) .

uninstall:
	rm -f $(INSTDIR)/$(BIN)

test:
	go test ./...

clean:
	rm -f $(BIN)
