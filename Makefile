.PHONY: help build install clean

BIN = git-chat
PREFIX ?= $(HOME)/.local

help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@echo "  build      Build the git-chat binary"
	@echo "  install    Build and install to $(PREFIX)/bin"
	@echo "  clean      Remove built and installed binaries"
	@echo "  help       Show this help"

build:
	go build -o $(BIN) .

install: build
	install -d $(PREFIX)/bin
	install $(BIN) $(PREFIX)/bin/$(BIN)
	@echo ""
	@echo "▸ git-chat installed to $(PREFIX)/bin/$(BIN)"
	@command -v $(BIN) >/dev/null 2>&1 && echo "▸ Ready — run '$(BIN)' to get started" || echo "▸ Note: $(PREFIX)/bin may not be on your PATH"

clean:
	rm -f $(BIN) $(PREFIX)/bin/$(BIN)

