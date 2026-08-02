# The web frontend, published only to a tailnet.
#
# Unlike the server image this cannot be distroless: the container runs
# tailscaled alongside the client, so it needs a shell for the entrypoint.
FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY . .
RUN CGO_ENABLED=0 go build -o /app/client ./cmd/client

FROM tailscale/tailscale:stable AS tailscale

FROM alpine:3.21
RUN apk add --no-cache ca-certificates iptables ip6tables

COPY --from=tailscale /usr/local/bin/tailscaled /usr/local/bin/tailscaled
COPY --from=tailscale /usr/local/bin/tailscale /usr/local/bin/tailscale
COPY --from=builder /app/client /app/client
COPY frontend/entrypoint.sh /app/entrypoint.sh
RUN chmod +x /app/entrypoint.sh

# No EXPOSE and no [[services]] in fly.toml: nothing is published to the
# public internet. The only way in is over the tailnet.
ENTRYPOINT ["/app/entrypoint.sh"]
