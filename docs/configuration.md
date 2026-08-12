---
title: "Configuration"
description: "Complete configuration reference for Cbox FPM Exporter including CLI flags, environment variables, and YAML"
weight: 4
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
| `--laravel` | Laravel site shorthand (`path` or `name:path`, repeatable) | - |
| `--laravel-site` | Laravel site property (`key=value`, repeatable) | - |
| `--laravel-config` | Path to a Laravel sites YAML file | - |

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
  --laravel-site enable_app_info=true \
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
| `CBOX_PHPFPM_ENABLED` | Enable PHP-FPM monitoring | `true` |
| `CBOX_PHPFPM_AUTODISCOVER` | Auto-discover pools | `true` |
| `CBOX_PHPFPM_RETRIES` | Discovery retry count | `5` |
| `CBOX_PHPFPM_RETRY_DELAY` | Delay between retries (seconds) | `2` |
| `CBOX_PHPFPM_POLL_INTERVAL` | Metrics poll interval | `1s` |
| `CBOX_PHP_ENABLED` | Enable PHP monitoring | `true` |
| `CBOX_PHP_BINARY` | PHP binary path | `php` |
| `CBOX_LOGGING_LEVEL` | Log level | `info` |
| `CBOX_LOGGING_FORMAT` | Log format (text, json) | `json` |
| `CBOX_LOGGING_COLOR` | Enable colored output | `true` |
| `CBOX_LARAVEL_SITES` | Laravel sites as a JSON array | - |
| `CBOX_LARAVEL_CONFIG` | Path to a Laravel sites YAML file | - |

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

php:
  enabled: true
  binary: /usr/bin/php

phpfpm:
  enabled: true
  autodiscover: true
  retries: 5
  retry_delay: 2
  poll_interval: 1s
  pools: []  # Manual pool config (see below)

laravel:
  - name: App
    path: /var/www/html
    enable_app_info: true
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
| `socket` | Main PHP-FPM socket (unix:// or tcp://) |
| `status_socket` | Separate socket for status (optional) |
| `status_path` | Path to status page (default: /status) |
| `config_path` | Path to pool config file |
| `binary` | PHP-FPM binary path |
| `cli_binary` | PHP CLI binary for this pool |
| `poll_interval` | Override global poll interval |
| `timeout` | Connection timeout |

## Laravel Configuration

Laravel configuration sources use the following precedence, from highest to lowest:

1. `--laravel-site`
2. `--laravel`
3. `CBOX_LARAVEL_SITES`
4. `--laravel-config`
5. `CBOX_LARAVEL_CONFIG`

Each site requires a unique `name`, an existing `path`, and an `artisan` file in that path.

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
  poll_interval: 5s

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

- [Metrics Reference](metrics-reference) - Understanding exported metrics
- [Laravel Monitoring](basic-usage/laravel-monitoring) - Detailed Laravel setup
