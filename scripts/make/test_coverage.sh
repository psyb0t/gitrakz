#!/bin/bash

set -euo pipefail

# gitrakz override of the framework's test_coverage.sh. The Makefile's
# find_script picks scripts/make/<name> ahead of scripts/make/servicepack/<name>,
# so this file wins. It is the framework script verbatim EXCEPT the coverage
# filter also drops *.gen.go: gitrakz commits oapi-codegen + gorm-gen output the
# vanilla framework has no concept of, and generated code must not count toward
# the coverage floor. Keep in step with the framework script on updates.

# Source common functions from the framework script dir alongside this override.
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/servicepack/common.sh"

MIN_TEST_COVERAGE=${MIN_TEST_COVERAGE:-90}

section "Running Tests with Coverage Check"
info "Running tests with coverage analysis..."

# Ensure cleanup on exit
trap 'rm -f coverage.txt coverage_filtered.txt' EXIT

# Run tests with coverage - need to use array for proper word splitting
readarray -t packages < <(go list ./... | grep -v /cmd | grep -v '/internal/pkg/services$' | grep -v /internal/pkg/services/)
if ! go test -race -coverprofile=coverage.txt "${packages[@]}"; then
	error "Tests failed"
	exit 1
fi

# Filter generated test doubles AND generated code (*.gen.go) from the aggregate
# while keeping ordinary test files in scope. awk is available in the Alpine dev
# image; GNU grep -P is not.
awk '!/internal\/pkg\/service-manager\/mocks\.go:/ && !/\.gen\.go:/' coverage.txt \
	>coverage_filtered.txt

coverage_summary=$(go tool cover -func=coverage_filtered.txt | awk '$1 == "total:" { print $3 }')
pct=${coverage_summary%%%}
integer_pct=${pct%%.*}

# Persist the decimal percentage for the badge pipeline (survives the trap above).
printf '%s\n' "$pct" >coverage-percent.txt

if [ -z "$pct" ]; then
	warning "No test coverage information available"

	exit 1
elif [ "$integer_pct" -lt "$MIN_TEST_COVERAGE" ]; then
	error "Coverage $pct% is less than the minimum $MIN_TEST_COVERAGE%"
	exit 1
else
	success "Coverage $pct% meets the minimum requirement of $MIN_TEST_COVERAGE%"
fi
