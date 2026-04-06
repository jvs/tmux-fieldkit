VERSION ?= dev

build:
	go build -ldflags "-s -w -X main.version=$(VERSION)" -o kit ./cmd/kit

install:
	go install ./cmd/kit

test:
	go test ./...

release:
	@if [ "$$(git branch --show-current)" != "main" ]; then \
		echo "error: not on main branch"; exit 1; \
	fi
	@if [ -n "$$(git status --porcelain)" ]; then \
		echo "error: working tree is dirty"; exit 1; \
	fi
	@if [ -z "$(VERSION)" ] || [ "$(VERSION)" = "dev" ]; then \
		echo "error: VERSION is required (e.g. make release VERSION=v0.1.0)"; exit 1; \
	fi
	git push origin main
	git tag $(VERSION)
	git push origin $(VERSION)

.PHONY: build install test release
