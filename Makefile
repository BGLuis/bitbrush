GOROOT := $(shell go env GOROOT)
WASM_OUT := web/public/main.wasm
WASM_EXEC := web/public/wasm_exec.js

.PHONY: wasm dev build test check clean

wasm:
	GOOS=js GOARCH=wasm go build -o $(WASM_OUT) ./cmd/wasm
	@if [ -f "$(GOROOT)/lib/wasm/wasm_exec.js" ]; then \
		cp "$(GOROOT)/lib/wasm/wasm_exec.js" $(WASM_EXEC); \
	else \
		cp "$(GOROOT)/misc/wasm/wasm_exec.js" $(WASM_EXEC); \
	fi

dev: wasm
	cd web && npm run dev

build: wasm
	cd web && npm run build

test:
	go test ./internal/...

check:
	go vet ./internal/...
	GOOS=js GOARCH=wasm go vet ./cmd/wasm
	gofmt -l .
	cd web && npm run check

clean:
	rm -f $(WASM_OUT) $(WASM_EXEC)
	rm -rf web/dist web/node_modules
