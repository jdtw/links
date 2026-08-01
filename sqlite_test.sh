#! /bin/bash
# Runs the full integration suite against the SQLite store. Unlike
# docker_test.sh and local_test.sh this needs no database server: the store is
# a file in a scratch directory that is removed on exit.
set -euxo pipefail

SQLITE_DIR="$(mktemp -d)"
export SQLITE_PATH="${SQLITE_DIR}/links.db"

cleanup() {
       exit_status=$?
       echo "Cleaning up ${SQLITE_DIR}..."
       rm -rf "${SQLITE_DIR}"
       exit "${exit_status}"
}
trap cleanup EXIT

go test ./...

./test.sh
