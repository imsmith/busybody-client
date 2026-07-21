.PHONY: build test

build:
	cd go && CGO_ENABLED=0 go build -o ../busybody .

test:
	cd go && go test ./...
