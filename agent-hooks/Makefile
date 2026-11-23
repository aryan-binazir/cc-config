BIN ?= $(HOME)/.local/bin/agent-hooks

.PHONY: build test check install uninstall

build:
	go build -o bin/agent-hooks .

test:
	go test ./...

check:
	@test -z "$$(gofmt -l .)" || { gofmt -l .; echo "run gofmt -w ."; exit 1; }
	go vet ./...
	go test ./...
	go test -count=1 -run TestBinaryLatency -v .
	go test ./... -run '^$$' -bench . -benchtime 200x

install: build
	install -D -m 0755 bin/agent-hooks $(BIN)
	$(BIN) install

uninstall:
	$(BIN) uninstall
