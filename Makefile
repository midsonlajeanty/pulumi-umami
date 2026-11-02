VERSION ?= $(shell sed -n 's/.*"\.": *"\([^"]*\)".*/\1/p' release/.manifest.json)
PROVIDER := pulumi-resource-umami
LDFLAGS := -s -w -X main.version=$(VERSION)
PLATFORMS := linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64
GO_SOURCES := $(shell find . -name '*.go' -not -path './sdk/*' -not -path './tests/*')

.PHONY: all build lint test schema sdk sdk-check dist e2e clean

all: build

build: bin/$(PROVIDER)

bin/$(PROVIDER): $(GO_SOURCES) go.mod go.sum release/.manifest.json
	go build -ldflags "$(LDFLAGS)" -o $@ ./cmd/$(PROVIDER)

lint:
	test -z "$$(gofmt -l . | grep -v '^sdk/')"
	go vet ./...
	golangci-lint run

test:
	go test ./...

schema: build
	pulumi package get-schema ./bin/$(PROVIDER) > schema.json

sdk: build
	rm -rf sdk/nodejs
	pulumi package gen-sdk ./bin/$(PROVIDER) --language nodejs --out sdk
	cd sdk/nodejs && npm install --no-audit --no-fund --no-package-lock && npx tsc && cp package.json ../../README.md bin/

sdk-check: schema sdk
	git diff --exit-code -- schema.json sdk/nodejs

dist: $(foreach platform,$(PLATFORMS),dist/$(PROVIDER)-v$(VERSION)-$(subst /,-,$(platform)).tar.gz)
	cd dist && sha256sum *.tar.gz > $(PROVIDER)-v$(VERSION)-checksums.txt

dist/$(PROVIDER)-v$(VERSION)-%.tar.gz: $(GO_SOURCES) go.mod go.sum
	$(eval os := $(word 1,$(subst -, ,$*)))
	$(eval arch := $(word 2,$(subst -, ,$*)))
	$(eval ext := $(if $(filter windows,$(os)),.exe,))
	mkdir -p dist/$*
	CGO_ENABLED=0 GOOS=$(os) GOARCH=$(arch) go build -ldflags "$(LDFLAGS)" -o dist/$*/$(PROVIDER)$(ext) ./cmd/$(PROVIDER)
	tar -czf $@ -C dist/$* $(PROVIDER)$(ext)
	rm -rf dist/$*

e2e: build sdk
	tests/e2e/run.sh

clean:
	rm -rf bin dist sdk/nodejs/bin sdk/nodejs/node_modules tests/e2e/node_modules
