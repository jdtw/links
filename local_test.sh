#!/bin/bash
set -euxo pipefail

DB='postgres://localhost/test'

cleanup() {
       exit_status=$?
       psql "${DB}" -c 'drop table if exists links'
       exit "${exit_status}"
}
trap cleanup EXIT

psql "${DB}" -a -c '\i links.sql'
DATABASE_URL="${DB}" ./test.sh
psql "${DB}" -c 'select * from links'