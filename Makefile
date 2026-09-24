TESTS_DIR := tests
VERSION_FILE := versions.txt
MASK :=

export GOWORK := off
export GOFLAGS := -mod=vendor

.PHONY: help build unit-test e2e-test test clear-tests bump-patch bump-minor bump-major _bump-commit vendor vendor-check

help:
	@grep -E '^[a-zA-Z_-]+:' Makefile | sed 's/:.*//' | sort -u

build:
	@set -eu; \
	v=$$(tr -d '[:space:]' < $(VERSION_FILE)); \
	u=$$(git remote get-url origin 2>/dev/null || true); \
	case "$$u" in \
		"")    o=local ;; \
		*://*) h=$${u#*://}; h=$${h#*@}; o="https://$${h%.git}" ;; \
		*:*)   h=$${u#*@};   o="https://$$(printf '%s' "$${h%.git}" | tr ':' '/')" ;; \
		*)     o=local ;; \
	esac; \
	if [ -f upstream.txt ]; then up=$$(tr -d '[:space:]' < upstream.txt); else up="$$o"; fi; \
	c=$$(git rev-parse --short HEAD 2>/dev/null || true); \
	CGO_ENABLED=0 go build -trimpath \
		-ldflags="-s -w -X main.version=$$v -X main.origin=$$o -X main.upstream=$$up -X main.commit=$$c -X main.channel=local" \
		-o kdbx-cli ./cmd/kdbx-cli; \
	echo "Built: ./kdbx-cli (v$$v)"

unit-test:
	go test $(if $(MASK),-run $(MASK)) ./...

e2e-test: build
	@set -e; \
	mkdir -p $(TESTS_DIR)/_root; \
	KDBX_CLI_E2E_ROOT="$$(pwd)/$(TESTS_DIR)/_root" \
	KDBX_CLI_E2E_KEEP=0 \
	go test -tags=e2e $(if $(MASK),-run $(MASK)) ./...; \
	status=$$?; \
	rm -rf $(TESTS_DIR)/_root; \
	exit $$status

test: unit-test e2e-test

clear-tests:
	@rm -rf $(TESTS_DIR)/*
	@touch $(TESTS_DIR)/.gitkeep
	@echo "Cleared $(TESTS_DIR)/"

bump-patch:
	@set -eu; \
	v=$$(tr -d '[:space:]' < $(VERSION_FILE)); \
	maj=$${v%%.*}; rest=$${v#*.}; min=$${rest%%.*}; pat=$${rest#*.}; \
	printf '%s.%s.%s\n' "$$maj" "$$min" "$$((pat + 1))" > $(VERSION_FILE); \
	cat $(VERSION_FILE)
	@$(MAKE) _bump-commit LEVEL=patch

bump-minor:
	@set -eu; \
	v=$$(tr -d '[:space:]' < $(VERSION_FILE)); \
	maj=$${v%%.*}; rest=$${v#*.}; min=$${rest%%.*}; \
	printf '%s.%s.0\n' "$$maj" "$$((min + 1))" > $(VERSION_FILE); \
	cat $(VERSION_FILE)
	@$(MAKE) _bump-commit LEVEL=minor

bump-major:
	@set -eu; \
	v=$$(tr -d '[:space:]' < $(VERSION_FILE)); \
	maj=$${v%%.*}; \
	printf '%s.0.0\n' "$$((maj + 1))" > $(VERSION_FILE); \
	cat $(VERSION_FILE)
	@$(MAKE) _bump-commit LEVEL=major

_bump-commit:
	@set -eu; \
	v=$$(tr -d '[:space:]' < $(VERSION_FILE)); \
	if git rev-parse -q --verify "refs/tags/v$$v" >/dev/null; then \
		git checkout -- $(VERSION_FILE); \
		echo "tag v$$v already exists" >&2; \
		exit 1; \
	fi; \
	git commit -q -m "bump $(LEVEL)" -- $(VERSION_FILE); \
	git tag "v$$v"; \
	rc=0; \
	for r in $$(git remote); do \
		git push -q "$$r" HEAD --tags || { echo "push to $$r failed" >&2; rc=1; }; \
	done; \
	echo "Tagged v$$v"; \
	exit "$$rc"

vendor:
	GOWORK=off go mod tidy
	GOWORK=off go mod vendor

vendor-check:
	GOWORK=off go mod vendor
	test -z "$$(git status --porcelain -- go.mod go.sum vendor/ | tee /dev/stderr)"
