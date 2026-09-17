BINARY  := unicd
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null | sed 's/^v//')
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
DATE    ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -s -w \
	-X 'github.com/denck/unicd/cmd.Version=$(VERSION)' \
	-X 'github.com/denck/unicd/cmd.Commit=$(COMMIT)' \
	-X 'github.com/denck/unicd/cmd.BuildDate=$(DATE)'

.PHONY: all build install fmt vet test tidy clean release

all: build

build:
	CGO_ENABLED=0 go build -trimpath -ldflags "$(LDFLAGS)" -o $(BINARY) .

install: build
	install -m 0755 $(BINARY) /usr/local/bin/$(BINARY)

fmt:
	gofmt -w .

vet:
	go vet ./...

test:
	go test ./...

tidy:
	go mod tidy

clean:
	rm -f $(BINARY) $(BINARY).exe
	rm -rf dist/

release:
	@mkdir -p dist
	@for target in linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64; do \
		os=$${target%/*}; arch=$${target#*/}; ext=""; \
		[ "$$os" = "windows" ] && ext=".exe"; \
		echo "building $$os/$$arch"; \
		GOOS=$$os GOARCH=$$arch CGO_ENABLED=0 go build -trimpath -ldflags "$(LDFLAGS)" \
			-o dist/$(BINARY)_$(VERSION)_$${os}_$${arch}$$ext . ; \
	done
