tests_dir := "tests"
version_file := "versions.txt"
export GOWORK := "off"
export GOFLAGS := "-mod=vendor"

_default:
    @just --list

build:
    #!/usr/bin/env sh
    set -eu
    v=$(tr -d '[:space:]' < {{version_file}})
    u=$(git remote get-url origin 2>/dev/null || true)
    case "$u" in
        "")    o=local ;;
        *://*) h=${u#*://}; h=${h#*@}; o="https://${h%.git}" ;;
        *:*)   h=${u#*@};   o="https://$(printf '%s' "${h%.git}" | tr ':' '/')" ;;
        *)     o=local ;;
    esac
    if [ -f upstream.txt ]; then up=$(tr -d '[:space:]' < upstream.txt); else up="$o"; fi
    c=$(git rev-parse --short HEAD 2>/dev/null || true)
    CGO_ENABLED=0 go build -trimpath \
        -ldflags="-s -w -X main.version=$v -X main.origin=$o -X main.upstream=$up -X main.commit=$c -X main.channel=local" \
        -o kdbx-cli ./cmd/kdbx-cli
    echo "Built: ./kdbx-cli (v$v)"

unit-test mask="":
    go test {{ if mask != "" { "-run " + mask } else { "" } }} ./...

e2e-test mask="": build
    #!/usr/bin/env sh
    set -e
    mkdir -p {{tests_dir}}/_root
    KDBX_CLI_E2E_ROOT="$(pwd)/{{tests_dir}}/_root" \
    KDBX_CLI_E2E_KEEP=0 \
    go test -tags=e2e {{ if mask != "" { "-run " + mask } else { "" } }} ./...; \
    status=$?; \
    rm -rf {{tests_dir}}/_root; \
    exit $status

test: unit-test e2e-test

clear-tests:
    @rm -rf {{tests_dir}}/*
    @touch {{tests_dir}}/.gitkeep
    @echo "Cleared {{tests_dir}}/"

bump-patch: && (_bump-commit "patch")
    #!/usr/bin/env sh
    set -eu
    v=$(tr -d '[:space:]' < {{version_file}})
    IFS=. read -r MAJ MIN PAT <<EOF
    $v
    EOF
    printf '%s.%s.%s\n' "$MAJ" "$MIN" "$((PAT + 1))" > {{version_file}}
    cat {{version_file}}

bump-minor: && (_bump-commit "minor")
    #!/usr/bin/env sh
    set -eu
    v=$(tr -d '[:space:]' < {{version_file}})
    IFS=. read -r MAJ MIN PAT <<EOF
    $v
    EOF
    printf '%s.%s.0\n' "$MAJ" "$((MIN + 1))" > {{version_file}}
    cat {{version_file}}

bump-major: && (_bump-commit "major")
    #!/usr/bin/env sh
    set -eu
    v=$(tr -d '[:space:]' < {{version_file}})
    IFS=. read -r MAJ MIN PAT <<EOF
    $v
    EOF
    printf '%s.0.0\n' "$((MAJ + 1))" > {{version_file}}
    cat {{version_file}}

_bump-commit level:
    #!/usr/bin/env sh
    set -eu
    v=$(tr -d '[:space:]' < versions.txt)
    if git rev-parse -q --verify "refs/tags/v$v" >/dev/null; then
        git checkout -- versions.txt
        echo "tag v$v already exists" >&2
        exit 1
    fi
    git commit -q -m "bump {{level}}" -- versions.txt
    git tag "v$v"
    rc=0
    for r in $(git remote); do
        git push -q "$r" HEAD --tags || { echo "push to $r failed" >&2; rc=1; }
    done
    echo "Tagged v$v"
    exit "$rc"

vendor:
    GOWORK=off go mod tidy
    GOWORK=off go mod vendor

vendor-check:
    GOWORK=off go mod vendor
    test -z "$(git status --porcelain -- go.mod go.sum vendor/ | tee /dev/stderr)"
