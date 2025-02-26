.PHONY: build serve clean

GOARCH=wasm
GOOS=js
WASM_FILE=public/main.wasm
WASM_EXEC=public/wasm_exec.js

build: $(WASM_FILE) $(WASM_EXEC)

$(WASM_FILE):
	GOOS=$(GOOS) GOARCH=$(GOARCH) go build -o $(WASM_FILE) ./wasm

$(WASM_EXEC):
	mkdir -p public
	cp "$$(go env GOROOT)/misc/wasm/wasm_exec.js" $(WASM_EXEC)

serve: build
	go run server/main.go

clean:
	rm -rf public 