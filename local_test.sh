#!/bin/bash
set -euxo pipefail

export DATABASE_URL='postgres://localhost/test'

cleanup() {
       exit_status=$?
       psql "${DATABASE_URL}" -c 'drop table if exists links'
       exit "${exit_status}"
}
trap cleanup EXIT

psql "${DATABASE_URL}" -a -c '\i links.sql'
./test.sh
psql "${DATABASE_URL}" -c 'select * from links'