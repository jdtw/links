# Tailnet-only web frontend

Runs `client --server` on Fly, published to a tailnet and nowhere else.

The frontend signs requests with a key that can add, edit and delete any
link, and it performs no authentication of its own -- anyone who can reach
it has full control. It must therefore never be exposed publicly. What
enforces that here is that the app has no public ingress at all: `fly.toml`
declares no services, so Fly allocates no public address and the app needs
no dedicated IP. `fly ips list` should be empty, and that is worth checking
after any change to the config.

Reachability comes entirely from Tailscale. Note that `tailscale serve` is
not the only way in: in userspace-networking mode tailscaled also forwards
inbound tailnet connections to local listeners, so the client is reachable
both at `https://links.<tailnet>.ts.net` (via Serve, with a real
certificate) and directly at `http://<tailnet-ip>:9099`. Both are confined
to the tailnet, but the second bypasses Serve, so do not treat Serve as a
security boundary -- the boundary is the tailnet.

## Setup

Create an **ephemeral, reusable** auth key in the Tailscale admin console
(Settings -> Keys). Reusable because every deploy replaces the machine's
filesystem and therefore its Tailscale state, so the key is used again on
each deploy. Ephemeral so the node from the previous deploy is cleaned up
automatically instead of leaving `links-1`, `links-2` behind.

```
$ fly apps create links-frontend
$ fly secrets import -a links-frontend <<EOF
TS_AUTHKEY=tskey-auth-...
LINKS_PRIVATE_KEY_B64=$(base64 -i ~/.config/links/client-priv.pb)
EOF
$ fly deploy . -c frontend/fly.toml
```

`fly secrets import` reads from stdin, so neither value appears in the
process table the way `fly secrets set` would.

Pass the repository root explicitly to `fly deploy`: it is the build
context, and the Dockerfile needs the Go sources. The `dockerfile` key in
`fly.toml` is resolved relative to `fly.toml` itself, not to that context.

Then confirm no public ingress was allocated:

```
$ fly ips list -a links-frontend    # expect no rows
```

The frontend is reachable from the tailnet at `https://links.<tailnet>.ts.net`.

## Configuration

| Variable | Where | Purpose |
| --- | --- | --- |
| `TS_AUTHKEY` | secret | Tailscale auth key, ephemeral and reusable |
| `LINKS_PRIVATE_KEY_B64` | secret | base64 of the client signing key |
| `LINKS_ADDR` | `fly.toml` | the links server to talk to |
| `TS_HOSTNAME` | `fly.toml` | tailnet hostname, defaults to `links` |
| `PORT` | `fly.toml` | local port the client listens on |

The signing key is written to `/run/links/priv.pb` at startup, on tmpfs, so
it never lands in the image or on a volume.

## Notes

`tailscaled` needs a real state directory, not `--state=mem:`. Serve
provisions a Let's Encrypt certificate and caches it under the state dir, so
in-memory state fails every TLS handshake with `no TailscaleVarRoot`. The
state lives on the machine's ephemeral root filesystem, which is fine: it
survives restarts and is discarded when the machine is replaced.

`tailscale status` is not a usable readiness probe for the daemon. It exits
non-zero while the node is stopped or unauthenticated -- exactly the state
tailscaled is in before logging in -- so the entrypoint waits for the
control socket instead.

The machine cannot scale to zero. Fly wakes stopped machines from its own
proxy, and tailnet traffic never passes through it, so a stopped machine is
simply unreachable.

`client --server` binds all interfaces, not just loopback. With no Fly
services declared there is no public route to it, but it is reachable over
Fly's private 6PN network from other apps in the same organization, and over
the tailnet directly as noted above. Adding a bind-address flag to the
client would narrow this to loopback and make Serve the only route in.
