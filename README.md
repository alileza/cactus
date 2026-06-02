# cactus

A minimal HTTP endpoint prober. Configure endpoints to watch, and get alerted via Telegram when something goes down or recovers.

## Features

- Probes any HTTP/HTTPS endpoint in parallel
- Per-probe interval, method, headers, and Basic Auth
- Follows redirects automatically
- Alerts on first failure and on recovery (no repeated noise)
- Telegram receiver (more can be added)
- Single static binary, no runtime dependencies

## Install

```sh
go install cactus@latest
```

Or pull the Docker image:

```sh
docker pull ghcr.io/alileza/cactus:latest
```

## Usage

```sh
cactus --config config.yaml
```

Docker:

```sh
docker run -v $(pwd)/config.yaml:/config.yaml ghcr.io/alileza/cactus --config /config.yaml
```

### Probing HTTPS endpoints

The image is built `FROM scratch` and ships a bundled CA store (since
the Dockerfile fix). If you're running an older image that predates
that change, you'll see every HTTPS probe fail with:

```
tls: failed to verify certificate: x509: certificate signed by unknown authority
```

Workaround for older images — mount the host's CA bundle in:

```sh
docker run \
  -v $(pwd)/config.yaml:/config.yaml \
  -v /etc/ssl/certs/ca-certificates.crt:/etc/ssl/certs/ca-certificates.crt:ro \
  ghcr.io/alileza/cactus --config /config.yaml
```

Or in `docker-compose.yml`:

```yaml
services:
  cactus:
    image: ghcr.io/alileza/cactus:latest
    volumes:
      - ./config.yaml:/config.yaml:ro
      - /etc/ssl/certs/ca-certificates.crt:/etc/ssl/certs/ca-certificates.crt:ro
```

The host path differs slightly across distros — `/etc/ssl/certs/ca-certificates.crt`
on Debian/Ubuntu, `/etc/ssl/cert.pem` on Alpine, `/etc/pki/tls/cert.pem` on
RHEL/Fedora. Mount whatever your host has at the `/etc/ssl/certs/ca-certificates.crt`
path inside the container (which is where Go's `crypto/tls` looks by default).

## Configuration

```yaml
probes:
  - name: public-api
    url: https://example.com/health
    method: GET
    interval: 30s
    expected_status: 200

  - name: protected-api
    url: https://example.com/admin
    method: GET
    interval: 60s
    auth:
      username: user
      password: pass
    expected_status: 200

  - name: post-endpoint
    url: https://example.com/ingest
    method: POST
    interval: 2m
    headers:
      Content-Type: application/json
    expected_status: 202

receivers:
  telegram:
    bot_token: "YOUR_BOT_TOKEN"
    chat_id: "YOUR_CHAT_ID"
```

| Field | Required | Default | Description |
|---|---|---|---|
| `name` | yes | — | Label used in alerts |
| `url` | yes | — | Endpoint to probe |
| `method` | no | `GET` | HTTP method |
| `interval` | no | `60s` | How often to probe |
| `expected_status` | no | `200` | HTTP status code considered healthy |
| `auth.username` | no | — | Basic Auth username |
| `auth.password` | no | — | Basic Auth password |
| `headers` | no | — | Extra request headers |

### Telegram setup

1. Create a bot via [@BotFather](https://t.me/botfather) and copy the token.
2. Add the bot to a group or start a DM, then get the chat ID from `https://api.telegram.org/bot<TOKEN>/getUpdates`.
3. Set `bot_token` and `chat_id` in the config.

## Release

Images are published to `ghcr.io/alileza/cactus` via GitHub Actions on manual dispatch.

```sh
# multi-arch (amd64 + arm64)
docker pull ghcr.io/alileza/cactus:v1.0.0

# amd64 only
docker pull ghcr.io/alileza/cactus:v1.0.0-amd64
```

## Development

```sh
go test ./...
go build -ldflags="-s -w" -o cactus .
```
