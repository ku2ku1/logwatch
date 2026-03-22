.PHONY: build run test clean build-arm64

build:
	go build -o bin/logwatch ./cmd/logwatch

# Cross-compile for Linux ARM64 from x86_64. Requires a proper AArch64 toolchain or the
# host assembler will choke on ARM64 instructions (stp x29,x30, ...).
# One-time on Debian/Kali:  sudo apt install gcc-aarch64-linux-gnu
build-arm64:
	CC=aarch64-linux-gnu-gcc CGO_ENABLED=1 GOOS=linux GOARCH=arm64 \
		go build -ldflags="-s -w" -o bin/logwatch-arm64 ./cmd/logwatch

run:
	go run ./cmd/logwatch

test:
	go test ./...

clean:
	rm -rf bin/ data/

tidy:
	go mod tidy

# Production targets
build-prod:
	cd frontend && npm run build
	cp -r frontend/dist internal/api/dist
	CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o bin/logwatch ./cmd/logwatch

deploy-local:
	sudo bash deploy/install.sh
