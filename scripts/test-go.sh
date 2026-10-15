#!/usr/bin/env bash
set -Eeuo pipefail

cd "$(dirname "${BASH_SOURCE[0]}")/.."
export GOTOOLCHAIN="${GOTOOLCHAIN:-auto}"

printf '\n== Go toolchain ==\n'
go version

printf '\n== Verify dependencies ==\n'
go mod download
go mod verify

printf '\n== Check gofmt ==\n'
unformatted="$(gofmt -l .)"
if [[ -n "$unformatted" ]]; then
  printf 'Files requiring gofmt:\n%s\n' "$unformatted" >&2
  exit 1
fi

printf '\n== Vet and unit tests ==\n'
make test

printf '\n== Race detector ==\n'
CGO_ENABLED=1 go test -race -count=1 ./...

printf '\n== Coverage ==\n'
go test -count=1 -coverprofile=coverage.out ./...
go tool cover -func=coverage.out | tail -n 1

printf '\n== Build and self-test ==\n'
make build
./dist/smokestack selftest

printf '\nGo test suite completed successfully.\n'
