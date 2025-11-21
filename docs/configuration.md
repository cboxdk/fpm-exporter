---
title: "Configuration"
description: "Complete configuration reference for PHPeek PHP-FPM Exporter including CLI flags, environment variables, and YAML"
weight: 4
---

# Configuration

PHPeek PHP-FPM Exporter supports three configuration methods with the following precedence:

1. **CLI Flags** (highest priority)
2. **Environment Variables**
3. **YAML Config File** (lowest priority)

## CLI Flags

```bash
phpeek-fpm-exporter serve [flags]
```

| Flag | Description | Default |
|------|-------------|---------|
| `--debug` | Enable debug mode | `false` |
| `--config` | Path to config file | - |
| `--autodiscover` | Auto-discover PHP-FPM pools | `true` |
| `--log-level` | Log level (debug, info, warn, error) | `info` |
| `--laravel` | Laravel site config (repeatable) | - |

### Laravel Flag Format

```bash
--laravel "name=SiteName,path=/path/to/laravel,connection=redis,queues=default|emails"
```

Parameters:
- `name` - Identifier for the site (default: "App")
- `path` - **Required** - Path to Laravel application root
- `connection` - Queue connection name (redis, database, sqs, etc.)
- `queues` - Pipe-separated list of queue names

Multiple sites:

```bash
phpeek-fpm-exporter serve \
  --laravel "name=Site1,path=/var/www/site1,connection=redis,queues=default" \
  --laravel "name=Site2,path=/var/www/site2,connection=database,queues=jobs|emails"
```

## Environment Variables

All environment variables use the `PHPEEK_` prefix:

| Variable | Description | Default |
|----------|-------------|---------|
| `PHPEEK_DEBUG` | Enable debug mode | `false` |
| `PHPEEK_MONITOR_LISTEN_ADDR` | Metrics listen address | `:9114` |
| `PHPEEK_MONITOR_ENABLE_JSON` | Enable JSON endpoint | `true` |
| `PHPEEK_PHPFPM_ENABLED` | Enable PHP-FPM monitoring | `true` |
| `PHPEEK_PHPFPM_AUTODISCOVER` | Auto-discover pools | `true` |
| `PHPEEK_PHPFPM_RETRIES` | Discovery retry count | `5` |
| `PHPEEK_PHPFPM_RETRY_DELAY` | Delay between retries (seconds) | `2` |
| `PHPEEK_PHPFPM_POLL_INTERVAL` | Metrics poll interval | `1s` |
| `PHPEEK_PHP_ENABLED` | Enable PHP monitoring | `true` |
| `PHPEEK_PHP_BINARY` | PHP binary path | `php` |
| `PHPEEK_LOGGING_LEVEL` | Log level | `info` |
| `PHPEEK_LOGGING_FORMAT` | Log format (text, json) | `json` |
| `PHPEEK_LOGGING_COLOR` | Enable colored output | `true` |

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
phpeek-fpm-exporter serve --config /path/to/config.yaml
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
phpeek-fpm-exporter serve
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
PHPEEK_DEBUG=true \
PHPEEK_LOGGING_LEVEL=debug \
PHPEEK_LOGGING_FORMAT=text \
phpeek-fpm-exporter serve \
  --laravel "name=Dev,path=/home/dev/app"
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
