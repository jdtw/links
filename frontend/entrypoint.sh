#!/bin/sh
# Brings up tailscaled, publishes the frontend to the tailnet, and runs the
# client. Nothing here listens on a public address: the container declares no
# Fly services, and tailscaled runs in userspace-networking mode so the only
# route in is Tailscale Serve.
set -eu

PORT="${PORT:-9099}"
TS_HOSTNAME="${TS_HOSTNAME:-links}"
SOCK=/run/tailscaled.sock

if [ -z "${TS_AUTHKEY:-}" ]; then
  echo "entrypoint: TS_AUTHKEY must be set" >&2
  exit 1
fi
if [ -z "${LINKS_PRIVATE_KEY_B64:-}" ]; then
  echo "entrypoint: LINKS_PRIVATE_KEY_B64 must be set" >&2
  exit 1
fi
if [ -z "${LINKS_ADDR:-}" ]; then
  echo "entrypoint: LINKS_ADDR must be set" >&2
  exit 1
fi

# The client takes a path to the signing key, so materialize it from the
# secret. /run is a tmpfs, so it never reaches the image or a volume.
mkdir -p /run/links
umask 077
echo "${LINKS_PRIVATE_KEY_B64}" | base64 -d > /run/links/priv.pb
export LINKS_PRIVATE_KEY=/run/links/priv.pb

# userspace-networking avoids needing a TUN device or NET_ADMIN.
#
# The state directory is required, not optional: `serve --https` provisions a
# Let's Encrypt certificate and caches it under the state dir, so running with
# --state=mem: fails every TLS handshake with "no TailscaleVarRoot". The
# machine has no volume, so this lives on the ephemeral root filesystem and is
# discarded when the machine is replaced -- which is why the auth key must be
# reusable, and ephemeral so the old node is cleaned up.
mkdir -p /var/lib/tailscale
/usr/local/bin/tailscaled \
  --tun=userspace-networking \
  --statedir=/var/lib/tailscale \
  --socket="${SOCK}" &
TAILSCALED_PID=$!

# Shut tailscaled down with the container so the ephemeral node deregisters
# promptly instead of lingering until it times out.
trap 'kill "${TAILSCALED_PID}" 2>/dev/null || true' INT TERM EXIT

# Wait for the daemon's control socket. Note that `tailscale status` is NOT a
# usable readiness probe here: it exits non-zero while the node is stopped or
# unauthenticated, which is exactly the state tailscaled is in before we log
# in, so polling it would never succeed.
i=0
until [ -S "${SOCK}" ]; do
  i=$((i + 1))
  if [ "${i}" -gt 30 ]; then
    echo "entrypoint: tailscaled socket never appeared" >&2
    exit 1
  fi
  sleep 1
done

/usr/local/bin/tailscale --socket="${SOCK}" up \
  --authkey="${TS_AUTHKEY}" \
  --hostname="${TS_HOSTNAME}"

# Publish https://<hostname>.<tailnet>.ts.net to the local client. Serve is
# what makes the frontend reachable at all in userspace-networking mode, and
# it is reachable only from within the tailnet.
/usr/local/bin/tailscale --socket="${SOCK}" serve --bg --https=443 "http://127.0.0.1:${PORT}"

echo "entrypoint: serving the links frontend to the tailnet as ${TS_HOSTNAME}"
exec /app/client --server "${PORT}"
