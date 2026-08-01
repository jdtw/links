# Dev Environment

This directory contains a docker compose file config that can be used to test locally end-to-end. Running `docker compose up` will:

- Start a links server on port 8080, backed by a SQLite database on the `links_data` volume.
- Start a links web client on port 9090.

The database persists across restarts in that volume. To start from an empty database, run `docker compose down -v`.

Add a link to the server using the frontend at http://localhost:9090.

Use the link via the server at http://localhost:8080.

The frontend authenticates to the server using hard-coded keys in this directory. Keys can be regenerated via the `keygen.sh` script.

> **Warning**
> Don't deploy this to production! The web frontend will sign any request, so access to it must be locked down. I use Tailscale for this purpose.

