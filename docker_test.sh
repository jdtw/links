#! /bin/bash
set -euxo pipefail

echo "Starting postgres container..."
docker-compose -f dev/docker-compose.yml up -d db

cleanup() {
       exit_status=$?
       echo "Cleaning up postgres container..."
       docker-compose -f dev/docker-compose.yml down --remove-orphans
       exit "${exit_status}"
}
trap cleanup EXIT

export DATABASE_URL="postgres://postgres:postgres@localhost:15432/postgres?sslmode=disable"

# Wait for postgres to be ready
until docker-compose -f dev/docker-compose.yml exec db pg_isready; do
    echo "Waiting for postgres to start..."
    sleep 1
done

./test.sh
