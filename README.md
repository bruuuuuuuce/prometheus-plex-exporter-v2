# Prometheus Exporter for Plex

Expose library playback, storage, and host metrics in a Prometheus format.

# Configuration

Configure the exporter with these environment variables:

- `PLEX_SERVER`: The full URL where your server can be reached, including the scheme and port (if not 80 or 443). For example `http://192.168.0.10:32400` or `https://my.plex.tld`.
- `PLEX_TOKEN`: A [Plex token](https://support.plex.tv/articles/204059436-finding-an-authentication-token-x-plex-token/) belonging to the server administrator. 
- `LOG_LEVEL` (optional): Minimum log level: `debug`, `info`, `warn`, or `error`. Defaults to `info`.

### Plex websocket notifications

The Plex notifications websocket exposes event types beyond playback, and Plex does not publish a complete schema for them. The exporter intentionally ignores the observed `progress`, `status`, and `provider.content.change` event types; they are not inputs to playback metrics. Other unrecognized event types are logged at debug level and can be inspected with `LOG_LEVEL=debug`. Event names are trimmed before dispatch so surrounding whitespace does not prevent recognized events such as `playing` from being handled.

The text `&#x20;` is the HTML numeric character reference for a space (U+0020). It is not emitted as such by the websocket logger. Its appearance in copied/aggregated log output points to HTML escaping in a display or export step, but without the original raw log record or websocket payload the exact stage cannot be identified. Plex's websocket event API is undocumented, so event names alone do not establish the payload semantics.

# Running

Building the exporter from source requires Go 1.25 or newer.

The exporter runs via Docker:

```bash
docker run \
  --name prom-plex-exporter \
  -p 9000:9000 \
  -e PLEX_SERVER="<Your Plex server URL>" \
  -e PLEX_TOKEN="<Your Plex server admin token>" \
  -e LOG_LEVEL="info" \
  bruuuuuuuce/prometheus-plex-exporter:latest
```

Or via Docker Compose:

```yaml
prom-plex-exporter:
  image: bruuuuuuuce/prometheus-plex-exporter:latest
  ports:
    - 9000:9000/tcp
  environment:
    PLEX_SERVER: <Your Plex server URL>
    PLEX_TOKEN: <Your Plex server admin token>
    LOG_LEVEL: info
```

The maintained image is published to [Docker Hub](https://hub.docker.com/r/bruuuuuuuce/prometheus-plex-exporter) for `linux/amd64` and `linux/arm64`. Images are built by this repository's [publish workflow](.github/workflows/publish.yml): every commit to `main` publishes `main` and an immutable `sha-<commit>` tag, while a Git tag such as `v1.2.3` publishes `v1.2.3` and advances `latest`. Pull requests never publish images.

For reproducible deployments, pin a release tag (or image digest) instead of `latest`. To migrate from the former `ghcr.io/jsclayton/prometheus-plex-exporter` image, replace only the image reference; `PLEX_SERVER`, `PLEX_TOKEN`, `LOG_LEVEL`, port `9000`, and the `/metrics` endpoint are unchanged. To roll back, restore the previously deployed release tag or digest and redeploy.

Maintainers must configure the `DOCKERHUB_USERNAME` and `DOCKERHUB_TOKEN` GitHub Actions secrets. `DOCKERHUB_TOKEN` should be a Docker Hub access token with permission to push `bruuuuuuuce/prometheus-plex-exporter`. Create a GitHub release tag matching `v*` only after CI passes on `main`; the tag triggers the versioned and `latest` publication.

A sample dashboard can be found in the [examples](examples/dashboards/Media%20Server.json)

# Exporting Metrics

Correcting the library labels on `plays_total` and `play_seconds_total` changes the identity of affected time series. After upgrading, Prometheus will store new samples under the corrected label values; historical samples remain under their previous values until they expire by retention.

The simplest way to start visualizaing your metrics is with the Free Forever [Grafana Cloud](https://grafana.com/docs/grafana-cloud/) and [Grafana Agent](https://grafana.com/docs/agent/latest/).

Here's an example config file that will read metrics from the exporter and ship them to [Prometheus](https://grafana.com/docs/grafana-cloud/data-configuration/metrics/metrics-prometheus/) via `remote_write`:


```yaml
metrics:
  configs:
  - name: prom-plex
    scrape_configs:
      - job_name: prom-plex
        static_configs:
          - targets:
            - <IP/address and port of the exporter endpoint>
    remote_write:
      - url: <Your Metrics instance remote_write endpoint>
        basic_auth:
          username: <Your Metrics instance ID>
          password: <Your Grafana.com API Key>
```
