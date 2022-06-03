#!/bin/bash
set -euxo pipefail

STAGING=$(mktemp -d)
cleanup() {
       exit_status=$?
       rm -rf "${STAGING}"
       exit "${exit_status}"
}
trap cleanup EXIT

GOBIN="${STAGING}" go install jdtw.dev/token/cmd/tokenpb@latest
"${STAGING}/tokenpb" gen-key --subject test --pub "${STAGING}/pub.pb" --priv "${STAGING}/priv.pb"
"${STAGING}/tokenpb" add-key --pub "${STAGING}/pub.pb" "${STAGING}/keyset.pb"

cp -f "${STAGING}/priv.pb" priv.pb
cp -f "${STAGING}/keyset.pb" keyset.pb
KEYSET=$(base64 -i "keyset.pb")
echo "KEYSET=${KEYSET}" > .env

# The private key is in the image, so rebuild...
docker compose build

echo "Success!"
