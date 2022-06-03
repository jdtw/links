# Links
[![Go](https://github.com/jdtw/links/actions/workflows/test.yml/badge.svg?branch=main)](https://github.com/jdtw/links/actions/workflows/test.yml)

This repository contains the suite of tools used to run a link redirection service. It contains:

* An HTTP server that performs redirects and exposes a REST API.
* A full-featured client, with several modes:
  * Command-line.
  * Web client.
  * Keybase chatbot.

The tooling is designed to run a single, locked down instance of the redirection service with a limited set of clients.

## Quickstart

Run `docker compose up` from the `dev/` directory to get a local instance running.

## Server

The server maintains a key-value store of friendly names to URI redirect templates. For example, `rfc -> https://datatracker.ietf.org/doc/html/rfc{0}` will redirect `GET /rfc/5280` to `https://datatracker.ietf.org/doc/html/rfc5280`. Try it out: [jdtw.us/rfc/5280](https://jdtw.us/rfc/5280).

Since this was written for personal use (i.e. not just anyone can add a link), the "database" is a simple mutex-protected in-memory `map`. Each DB entry is written to a file when added (one file per key-value pair) via the API below, and all DB entries are read on service startup.

## REST API

* `GET /api/links` returns all links in the database.
  * Request body: empty
  * Response body: serialized `links.Links` proto.
  * Returns: 200 (OK)
* `GET /api/links/{link}` looks up a single link.
  * Request body: empty
  * Response body: serialized `links.Link` proto.
  * Returns: 200 (OK) or 404 (not found)
* `PUT /api/links/{link}` creates or updates a link.
  * Request body: serialized `links.Link` proto.
  * Response body: empty
  * Returns: 201 (created) if created, or 204 (no content) if updated.
* `DELETE /api/links/{link}` removes a link.
  * Request body: empty
  * Response body: empty
  * Returns: 204 (no content)

All API endpoints require authentication via a [token](https://github.com/jdtw/token).

## Authentication

Authentication is done via signed proto [tokens](https://github.com/jdtw/token). Clients have a private Ed25519 key for signing them, and the server has a keyset of verification keys. Providing a client with a signing key directly is not standard, but since I control all of the clients for my use case, as well as the verification keyset that the server is provisioned with, it is nice not to have to go through an auth flow.

## Client

The client tool uses a private key to sign tokens for itself and authenticate to the REST API outlined above. The client can run in three different modes:
1. Command line.
1. HTTP server.
1. Keybase bot.

In any mode, the client requires a path to the private key and the address of the HTTPS enpoint hosting the REST API. These can be provided by command line flags (`--priv` and `--addr`, respectively), or by using the `LINKS_PRIVATE_KEY` and `LINKS_ADDR` environment variables.

> **Note:**
> The examples below assume that the `LINKS_PRIVATE_KEY` and `LINKS_ADDR` environment variables are set.

### Command line client

When run with no arguments, lists all links:
```
$ client
```

Add a link:
```
$ client --add=example --link=https://example.com
```

Get the redirect for a link:
```
$ client --get=example
```

Delete a link:
```
$ client --rm=example
```

### HTTP Frontend

Run an HTTP frontend on port 9999:
```
$ client --server=9999
```

This will expose a simple form that can be used to add and list links.

> **Warning**
> *DO NOT* expose this to the public internet unless you want to allow arbitrary access to add and view links. (I am currently running this web client exposed to my Tailscale network.)

### Keybase Chat Bot

Start a chat bot (requires keybase to be installed):
```
$ client --keybase_loc=/usr/bin/keybase
```

You can then chat with the keybase user running the client, using the following commands:

Add a link:
```
> !links add example https://example.com
```

List all links:
```
> !links ls
```

Get a specific link:
```
> !links ls example
```

Remove a link:
```
> !links rm example
```