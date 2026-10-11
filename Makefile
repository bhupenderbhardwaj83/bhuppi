.PHONY: build test cross install

export CGO_ENABLED=0

build:
	mkdir -p dist
	go build -trimpath -ldflags "-s -w" -o dist/bhuppi ./cmd/bhuppi

test:
	go test ./...

cross:
	mkdir -p dist
	GOOS=linux GOARCH=amd64 go build -trimpath -ldflags "-s -w" -o dist/bhuppi-linux-amd64 ./cmd/bhuppi
	GOOS=linux GOARCH=arm64 go build -trimpath -ldflags "-s -w" -o dist/bhuppi-linux-arm64 ./cmd/bhuppi
	GOOS=darwin GOARCH=amd64 go build -trimpath -ldflags "-s -w" -o dist/bhuppi-darwin-amd64 ./cmd/bhuppi
	GOOS=darwin GOARCH=arm64 go build -trimpath -ldflags "-s -w" -o dist/bhuppi-darwin-arm64 ./cmd/bhuppi
	GOOS=windows GOARCH=amd64 go build -trimpath -ldflags "-s -w" -o dist/bhuppi-windows-amd64.exe ./cmd/bhuppi
	GOOS=windows GOARCH=arm64 go build -trimpath -ldflags "-s -w" -o dist/bhuppi-windows-arm64.exe ./cmd/bhuppi

install:
	./install.sh
