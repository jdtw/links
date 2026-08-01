# Tailnet-only web frontend

Runs `client --server` on Fly, published to a tailnet and nowhere else.

The frontend signs requests with a key that can add, edit and delete any
link, and it performs no authentication of its own -- anyone who can reach
it has full control. It must therefore never be exposed publicly. Two things
enforce that here:

- `fly.toml` declares no services, so Fly allocates no public ingress. The
  app should have no dedicated IP; `fly ips list` should be empty.
- `tailscaled` runs in userspace-networking mode, so the container has no
  network interface reachable from outside. Tailscale Serve is the only
  path in, and it only accepts connections from the tailnet.

## Setup

Create an **ephemeral, reusable** auth key in the Tailscale admin console
(Settings -> Keys). Ephemeral matters: the container keeps no Tailscale
state on disk, so each start registers a new node, and ephemeral nodes are
removed automatically when they go offline instead of accumulating.

```
$ fly apps create links-frontend
$ fly secrets set -a links-frontend \
    TS_AUTHKEY="tskey-auth-..." \
    LINKS_PRIVATE_KEY_B64="$(base64 -i ~/.config/links/client-priv.pb)"
$ fly deploy -c frontend/fly.toml
```

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

The machine cannot scale to zero. Fly wakes stopped machines from its own
proxy, and tailnet traffic never passes through it, so a stopped machine is
simply unreachable.

`client --server` binds all interfaces, not just loopback. With no Fly
services declared there is no public route to it, but it is reachable over
Fly's private 6PN network from other apps in the same organization. Adding a
bind-address flag to the client would close that gap.
