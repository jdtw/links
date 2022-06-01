#! /bin/bash
set -euxo pipefail

TEST_DIR="$(pwd)/testdir"
PORT=9090
ADDR="http://localhost:${PORT}"

cleanup() {
       exit_status=$?
       rm -rf "${TEST_DIR}"
       killall -u ${USER} links || true
       exit "${exit_status}"
}
trap cleanup EXIT

mkdir "${TEST_DIR}"

go build -o "${TEST_DIR}" ./...
GOBIN="${TEST_DIR}" go install jdtw.dev/token/cmd/tokenpb@latest

KEYSET="${TEST_DIR}/ks.pb"
PUB="${TEST_DIR}/pub.pb"
PRIV="${TEST_DIR}/priv.pb"

"${TEST_DIR}/tokenpb" gen-key --subject "test" --pub "${PUB}" --priv "${PRIV}"
"${TEST_DIR}/tokenpb" add-key --pub "${PUB}" "${KEYSET}"
"${TEST_DIR}/tokenpb" dump-keyset "${KEYSET}"

mkdir "${TEST_DIR}/db"
DATABASE_URL='postgresql://localhost' "${TEST_DIR}/links" --port "${PORT}" \
        --keyset "${KEYSET}" &

until curl -s "${ADDR}"; do
    echo "Waiting for server to start..."
    sleep 1
done

TEST_OUTPUT='%{http_code} %{redirect_url}'

echo "Testing set index..."
"${TEST_DIR}/client" --priv "${PRIV}" \
         --addr "${ADDR}" \
         --index "http://example.com"
result=$(curl -s "${ADDR}" -o /dev/null -w "${TEST_OUTPUT}")
test "${result}" = "302 http://example.com/"

echo "Testing add redirect..."
"${TEST_DIR}/client" --priv "${PRIV}" \
         --addr "${ADDR}" \
         --add "foo" \
         --link "http://www.example.com"
result=$(curl -s "${ADDR}/foo" -o /dev/null -w "${TEST_OUTPUT}")
test "${result}" = "302 http://www.example.com/"

echo "Testing hyphens..."
result=$(curl -s "${ADDR}/f-o-o" -o /dev/null -w "${TEST_OUTPUT}")
test "${result}" = "302 http://www.example.com/"

echo "Testing get redirect..."
"${TEST_DIR}/client" --priv "${PRIV}" \
         --addr "${ADDR}" \
         --get "foo"

echo "Testing delete redirect..."
"${TEST_DIR}/client" --priv "${PRIV}" \
         --addr "${ADDR}" \
         --rm "foo"
result=$(curl -s "${ADDR}/foo" -o /dev/null -w "%{http_code}")
test "${result}" = "404"

echo "Testing redirect with param expansion..."
"${TEST_DIR}/client" --priv "${PRIV}" \
         --addr "${ADDR}" \
         --add "foo" \
         --link "http://www.example.com/bar/{0}"
result=$(curl -s "${ADDR}/foo/baz/quux" -o /dev/null -w "${TEST_OUTPUT}")
test "${result}" = "302 http://www.example.com/bar/baz/quux"

echo "Testing failed authorization..."
"${TEST_DIR}/tokenpb" gen-key --subject "untrusted" --pub "${TEST_DIR}/untrustedpub.pb" --priv "${TEST_DIR}/untrustedpriv.pb"
result=$("${TEST_DIR}/client" --priv "${TEST_DIR}/untrustedpriv.pb" \
                  --addr "${ADDR}" \
                  --add "evil" \
                  --link "http://www.example.com/evil" &&\
             echo "succeeded" || echo "failed")
test "${result}" = "failed"

echo "Testing nonce reuse..."
token=$("${TEST_DIR}/tokenpb" sign-token \
            --resource "GET localhost:9090/api/links" \
            --lifetime "2m" \
            "${PRIV}")
result=$(curl -s -H "Authorization: ${token}" \
     -o /dev/null \
     -w "%{http_code}" \
     "${ADDR}/api/links")
test "${result}" = "200"
result=$(curl -s -H "Authorization: ${token}" \
     -o /dev/null \
     -w "%{http_code}" \
     "${ADDR}/api/links")
test "${result}" = "401"

echo "Tests passed!"