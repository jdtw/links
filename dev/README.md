# Dev Environment

This directory contains a docker compose file config that can be used to test locally end-to-end. Running `docker compose up` will:

- Start postgres.
- Start a links server on port 8080.
- Start a links web client on port 9090.

Connect directly to the postgres instance using `postgres://postgres:postgres@localhost:15432/postgres`, or use the `psql.sh` script. Note that port is 15432 so that it doesn't conflict with any local postgres installation.

Add a link to the server using the frontend at http://localhost:9090.

Use the link via the server at http://localhost:8080.

The frontend authenticates to the server using hard-coded keys in this directory. Keys can be regenerated via the `keygen.sh` script.

> **Warning**
> Don't deploy this to production! The web frontend will sign any request, so access to it must be locked down. I use Tailscale for this purpose.

