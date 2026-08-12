---
title: "Reference"
description: "Complete configuration reference for Cbox FPM Exporter including CLI flags, environment variables, and YAML"
weight: 1
---

# Configuration

Cbox FPM Exporter supports three configuration methods with the following precedence:

1. **CLI Flags** (highest priority)
2. **Environment Variables**
3. **YAML Config File** (lowest priority)

## CLI Flags

```bash
fpm-exporter serve [flags]
```

| Flag | Description | Default |
|------|-------------|---------|
| `--debug` | Enable debug mode | `false` |
| `--config` | Path to config file | - |
| `--autodiscover` | Auto-discover PHP-FPM pools | `true` |
| `--log-level` | Log level (debug, info, warn, error) | `info` |
| `--laravel` | Laravel site shorthand (`path` or `name:path`, single site) | - |
| `--laravel-site` | Laravel site property (`key=value`, repeatable) | - |
| `--laravel-config` | Path to a Laravel sites YAML file | - |
| `--web.listen-address` | Address to serve metrics on (overrides `monitor.listen_addr`) | - |

### Laravel Flag Format

For simple sites, use the shorthand syntax:

```bash
# Path only (name defaults to "App")
--laravel /path/to/laravel

# Name and path
--laravel SiteName:/path/to/laravel
```

For queue monitoring and other options, use repeatable `--laravel-site` flags:

```bash
fpm-exporter serve \
  --laravel-site name=SiteName \
  --laravel-site path=/path/to/laravel \
  --laravel-site appinfo=true \
  --laravel-site queues.redis=default,emails
```

Multiple sites:

```bash
fpm-exporter serve \
  --laravel-site name=Site1 \
  --laravel-site path=/var/www/site1 \
  --laravel-site queues.redis=default \
  --laravel-site name=Site2 \
  --laravel-site path=/var/www/site2 \
  --laravel-site queues.database=jobs,emails
```

For complex setups, load the `laravel` section from a dedicated YAML file:

```bash
fpm-exporter serve --laravel-config /etc/cbox/laravel-sites.yaml
```

## Environment Variables

All environment variables use the `CBOX_` prefix:

| Variable | Description | Default |
|----------|-------------|---------|
| `CBOX_DEBUG` | Enable debug mode | `false` |
| `CBOX_MONITOR_LISTEN_ADDR` | Metrics listen address | `:9114` |
| `CBOX_MONITOR_ENABLE_JSON` | Enable JSON endpoint | `true` |
| `CBOX_MONITOR_SCRAPE_TIMEOUT` | Budget for one collection | `8s` |
| `CBOX_PHPFPM_ENABLED` | Enable PHP-FPM monitoring | `true` |
| `CBOX_PHPFPM_AUTODISCOVER` | Auto-discover pools | `true` |
| `CBOX_PHPFPM_RETRIES` | Discovery retry count | `5` |
| `CBOX_PHPFPM_RETRY_DELAY` | Delay between retries (seconds) | `2` |
| `CBOX_PHP_BINARY` | PHP binary path | `php` |
| `CBOX_LOGGING_LEVEL` | Log level | `info` |
| `CBOX_LOGGING_FORMAT` | Log format (text, json) | `json` |
| `CBOX_LOGGING_COLOR` | Enable colored output | `true` |
| `CBOX_LARAVEL_SITES` | Laravel sites as a JSON array | - |
| `CBOX_LARAVEL_CONFIG` | Path to a Laravel sites YAML file | - |

Every variable in this table is covered by a test asserting it actually changes
the effective configuration, so this table and the code cannot drift apart.

Example:

```bash
export CBOX_LARAVEL_SITES='[{"name":"App","path":"/var/www/html","queues":{"redis":["default"]}}]'
fpm-exporter serve
```

## YAML Configuration

Create a `config.yaml` file:

```yaml
debug: false

logging:
  level: info      # debug, info, warn, error
  format: json     # text, json
  color: true

monitor:
  listen_addr: ":9114"
  enable_json: true
  scrape_timeout: 8s    # Must stay below Prometheus' own scrape_timeout (10s by default)

php:
  binary: /usr/bin/php

phpfpm:
  enabled: true
  autodiscover: true
  retries: 5
  retry_delay: 2
  pools: []  # Manual pool config (see below)

laravel:
  - name: App
    path: /var/www/html
    enable_app_info: true
    timeout: 5s         # Per-site limit on each artisan call
    queues:
      redis:
        - default
        - emails
      database:
        - jobs
```

Use with:

```bash
fpm-exporter serve --config /path/to/config.yaml
```

## Manual Pool Configuration

Disable autodiscovery and configure pools manually:

```yaml
phpfpm:
  autodiscover: false
  pools:
    - socket: "unix:///var/run/php-fpm.sock"
      status_path: /status

    - socket: "tcp://127.0.0.1:9000"
      status_socket: "tcp://127.0.0.1:9001"  # Separate status socket
      status_path: /status
      timeout: 5s
```

### Pool Configuration Options

| Option | Description |
|--------|-------------|
| `name` | Pool name used in metric labels, and the label a pool keeps while it is unreachable |
| `socket` | Main PHP-FPM socket (unix:// or tcp://) |
| `status_socket` | Separate socket for status (optional; defaults to `socket`) |
| `status_path` | Path to the status page (default `/status`) |
| `config_path` | Path to pool config file |
| `binary` | PHP-FPM binary path |
| `cli_binary` | PHP CLI binary for this pool |
| `timeout` | Dial timeout for this pool (default 3s) |

## Laravel Configuration

Laravel configuration sources use the following precedence, from highest to lowest:

1. `--laravel-site`
2. `--laravel`
3. `CBOX_LARAVEL_CONFIG` (ignored when `--laravel-config` is set)
4. `CBOX_LARAVEL_SITES`
5. `--laravel-config`

Sites are merged by `name`: a site from a higher-precedence source replaces the
whole site with the same name from a lower-precedence one — the fields are not
merged individually.

Each site requires a unique `name`, an existing `path`, and an `artisan` file in that path.

### `--laravel-site` keys

| Key | Description |
|-----|-------------|
| `name` | Site identifier. A second `name=` starts a new site |
| `path` | Path to the Laravel application root |
| `appinfo` | `true`/`1` to collect `php artisan about` metrics |
| `queues.<connection>` | Comma-separated queue names for that connection |

Note that the flag uses `appinfo`, while the YAML and JSON sources use
`enable_app_info`. Any other key is rejected with an error.

### Basic Setup

```yaml
laravel:
  - name: MyApp
    path: /var/www/html
```

### With Queue Monitoring

```yaml
laravel:
  - name: MyApp
    path: /var/www/html
    queues:
      redis:
        - default
        - high
        - low
      database:
        - notifications
```

### With Custom PHP Binary

```yaml
laravel:
  - name: LegacyApp
    path: /var/www/legacy
    php_config:
      binary: /usr/bin/php7.4
    queues:
      database:
        - jobs
```

## Configuration Examples

### Minimal (Auto-discover)

```bash
fpm-exporter serve
```

### Production Setup

```yaml
debug: false

logging:
  level: warn
  format: json

monitor:
  listen_addr: ":9114"

phpfpm:
  autodiscover: true

laravel:
  - name: Production
    path: /var/www/app
    queues:
      redis:
        - default
        - emails
        - notifications
```

### Development Setup

```bash
CBOX_DEBUG=true \
CBOX_LOGGING_LEVEL=debug \
CBOX_LOGGING_FORMAT=text \
fpm-exporter serve \
  --laravel Dev:/home/dev/app
```

### Multiple Applications

```yaml
laravel:
  - name: API
    path: /var/www/api
    queues:
      redis: [default, webhooks]

  - name: Admin
    path: /var/www/admin
    queues:
      database: [reports, exports]

  - name: Worker
    path: /var/www/worker
    queues:
      redis: [jobs, notifications, emails]
```

## Next Steps

- [Metrics Reference](../reference/metrics) - Understanding exported metrics
- [Laravel Monitoring](../basic-usage/laravel-monitoring) - Detailed Laravel setup
