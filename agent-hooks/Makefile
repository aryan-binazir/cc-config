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

# Check the name and the embedded hooks.yaml before replacing the installed binary.
install: build
	test "$$(basename "$(BIN)")" = agent-hooks
	echo '{}' | ./bin/agent-hooks pre
	install -D -m 0755 bin/agent-hooks "$(BIN)"
	"$(BIN)" install

uninstall:
	"$(BIN)" uninstall
