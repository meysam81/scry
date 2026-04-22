# Convenience targets. The authoritative build commands live in cmd/*/
# (for the Go CLI) and extension/package.json (for the browser extension).

.PHONY: all cli wasm extension extension-dev extension-clean test clean help

help:
	@echo "scry — available targets:"
	@echo "  make cli            Build the Go CLI → ./scry"
	@echo "  make wasm           Build the Go WASM bundle into extension/public/"
	@echo "  make extension      Full production build of the Chrome extension"
	@echo "  make extension-dev  Start the CRXJS dev server"
	@echo "  make test           Run Go tests"
	@echo "  make clean          Remove build artefacts"

cli:
	go build -o scry .

wasm:
	cd extension && bun run build:wasm

extension: wasm
	cd extension && bun run build:vite

extension-dev:
	cd extension && bun run dev

extension-clean:
	rm -rf extension/dist extension/public/scry.wasm extension/public/wasm_exec.js

test:
	go test ./core/... ./internal/...

clean: extension-clean
	rm -f scry
