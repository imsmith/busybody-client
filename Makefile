.PHONY: build test test-sh

build:
	cd go && CGO_ENABLED=0 go build -o ../busybody .

test:
	cd go && go test ./...

test-sh:
	sh/test-smoke.sh
	sh/test-exit-code.sh
