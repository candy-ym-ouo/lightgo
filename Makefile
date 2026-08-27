APP := lightgo-server
GOFILES := $(shell find . -name '*.go' -type f)

.PHONY: fmt test vet build check smoke package clean stats

fmt:
	gofmt -w $(GOFILES)

test:
	go test -race ./...

vet:
	go vet ./...

build:
	mkdir -p bin
	go build -trimpath -o bin/$(APP) ./cmd/server

check: fmt vet test build

smoke:
	./scripts/smoke.sh

package: check
	./scripts/package.sh

stats:
	@find . -name '*.go' ! -name '*_test.go' -type f | sort | xargs wc -l
	@printf 'files: '; find . -name '*.go' ! -name '*_test.go' -type f | wc -l | tr -d ' '

clean:
	rm -rf bin dist
