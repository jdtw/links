#! /bin/bash
set -euxo pipefail

TEST_DIR="./testtmp"
PORT=9090
ADDR="http://localhost:${PORT}"

killall -u ${USER} links || true
rm -rf "${TEST_DIR}"
mkdir "${TEST_DIR}"

go build -o . ./...

./auth --new --keyset "${TEST_DIR}/keyset.pb" \
       --priv "${TEST_DIR}/priv.pb" \
       --subject "test"
./auth --dump --keyset "${TEST_DIR}/keyset.pb"

mkdir "${TEST_DIR}/db"
./links --port "${PORT}" \
        --keyset "${TEST_DIR}/keyset.pb" \
        --database "${TEST_DIR}/db" &

until curl -s "${ADDR}"; do
    echo "Waiting for server to start..."
    sleep 1
done

TEST_OUTPUT='%{http_code} %{redirect_url}'

echo "Testing set index..."
./client --priv "${TEST_DIR}/priv.pb" \
         --addr "${ADDR}" \
         --index "http://example.com"
result=$(curl -s "${ADDR}" -o /dev/null -w "${TEST_OUTPUT}")
test "${result}" = "302 http://example.com/"

echo "Testing add redirect..."
./client --priv "${TEST_DIR}/priv.pb" \
         --addr "${ADDR}" \
         --add "foo" \
         --link "http://www.example.com"
result=$(curl -s "${ADDR}/foo" -o /dev/null -w "${TEST_OUTPUT}")
test "${result}" = "302 http://www.example.com/"

echo "Testing get redirect..."
./client --priv "${TEST_DIR}/priv.pb" \
         --addr "${ADDR}" \
         --get "foo"

echo "Testing delete redirect..."
./client --priv "${TEST_DIR}/priv.pb" \
         --addr "${ADDR}" \
         --rm "foo"
result=$(curl -s "${ADDR}/foo" -o /dev/null -w "%{http_code}")
test "${result}" = "404"

echo "Testing failed authorization..."
./auth --new --keyset "${TEST_DIR}/keyset.pb" \
       --priv "${TEST_DIR}/untrusted.pb" \
       --subject "untrusted-test"
result=$(./client --priv "${TEST_DIR}/untrusted.pb" \
                  --add "evil" \
                  --link "http://www.example.com/evil" &&\
             echo "succeeded" || echo "failed")
test "${result}" = "failed"

echo "Stopping test server..."
killall -u ${USER} links
