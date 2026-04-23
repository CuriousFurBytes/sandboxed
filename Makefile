BIN       := sbx
CMD       := ./cmd/sbx
VERSION   ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS   := -ldflags "-X main.version=$(VERSION)"
INSTALL   := $(HOME)/.local/bin

.PHONY: build test install clean lint doctor

build:
	go build $(LDFLAGS) -o $(BIN) $(CMD)

test:
	go test ./... -race -count=1

install: build
	install -Dm755 $(BIN) $(INSTALL)/$(BIN)
	@echo "installed to $(INSTALL)/$(BIN)"

clean:
	rm -f $(BIN)

lint:
	go vet ./...
	@command -v staticcheck >/dev/null && staticcheck ./... || true

doctor: install
	$(INSTALL)/$(BIN) doctor
