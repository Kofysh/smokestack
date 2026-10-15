# Testing the fork

This fork uses a dedicated ci-tests branch for additional Go checks, kept
separate from PR #89 and the upstream security-sensitive workflow files.

## GitHub Actions

The workflow in .github/workflows/go-tests.yml runs after every push to
ci-tests. It uses Ubuntu 24.04 and Go 1.26, then runs scripts/test-go.sh.

On some forks, Actions must first be enabled in Settings > Actions > General.
Check the Actions tab for the Fork Go tests workflow and its job output.
A coverage.out artifact is uploaded when the coverage stage finishes.

## Local Go 1.26

From the repository root:

~~~bash
bash scripts/test-go.sh
~~~

The script downloads and verifies modules, checks gofmt, runs make test
(go vet and all unit tests), runs the race detector, writes a coverage
report, builds the binary, and runs the built-in selftest.

## Docker

With Docker installed, from the repository root:

~~~bash
docker run --rm \
  -v "$PWD:/workspace" \
  -v smokestack-go-mod:/go/pkg/mod \
  -w /workspace \
  golang:1.26-bookworm \
  bash scripts/test-go.sh
~~~

The first run needs network access for the image and Go modules. Race
detection needs a C compiler, which the GitHub-hosted runner provides.

This independent workflow cannot approve or bypass the original
project's first-time-contributor review and CI requirements.
