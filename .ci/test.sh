#!/usr/bin/env bash

# test.sh - CI script for running all tests for the cluster-api-provider-cox.
#
# Parameters:
# - GO_VERSION      Version of Go to use for testing. (default: 1.17.6)
#
# Examples:
# - `USE_SYSTEM_GO=1 ./test.sh`: To test the script locally without gimme

set -o nounset
set -o errexit
set -o pipefail

project_root=$(realpath "$(dirname $0)/..")
GO_VERSION=${GO_VERSION:-1.17.6}

main() {
  # Move to the project directory
  pushd "${project_root}"
  trap on_exit EXIT

  # Setup simple logging
  set -x

  # Configure go
  configure_go

  # Run all tests
  # TODO(erwin) re-add `make test` when the tests can run without requiring
  #  access to Cox: https://github.com/platform9/cluster-api-provider-cox/issues/47
  make verify
}

on_exit() {
  ret=$?
  echo "-------cleanup--------"
  make clean || true
  popd
  exit ${ret}
}

configure_go() {
  if [ -n "${USE_SYSTEM_GO:-}" ] ; then
    echo "\$USE_SYSTEM_GO set, using system go instead of gimme"
    return 0
  else
    which gimme > /dev/null || (echo "error: missing required command 'gimme'" && exit 1)
    eval "$(GIMME_GO_VERSION=${GO_VERSION} gimme)"
  fi
  which go
  go version
}

main $@