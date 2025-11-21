# PHPeek PHP-FPM Exporter

PHPeek PHP-FPM Exporter is a lightweight, Go-based Prometheus exporter for PHP-FPM and Laravel applications.

## Features

- 📊 PHP-FPM metrics via FastCGI (using [fcgx](https://github.com/gophpeek/fcgx))
- ⚙️ Automatic PHP-FPM pool discovery via `php-fpm -tt`
- 🧠 Opcache statistics per FPM pool
- 🚦 Laravel queue sizes, app info, cache state
- 🔌 Prometheus `/metrics` endpoint + JSON at `/json`
- 🐘 Multi-site Laravel support

## Documentation

📖 **Full documentation available at [phpeek.com/docs/phpeek-fpm-exporter](https://phpeek.com/docs/phpeek-fpm-exporter/v1/introduction)**

Or browse the [docs folder](./docs) for:
- [Installation](./docs/installation.md)
- [Quickstart](./docs/quickstart.md)
- [Configuration](./docs/configuration.md)
- [Metrics Reference](./docs/metrics-reference.md)

## Quickstart

```bash
# Install (auto-detects OS/arch)
curl -fsSL https://raw.githubusercontent.com/gophpeek/phpeek-fpm-exporter/main/install.sh | sh

# Run with auto-discovery
phpeek-fpm-exporter serve

# With Laravel monitoring (shorthand)
phpeek-fpm-exporter serve --laravel App:/var/www/html

# With Laravel queues (explicit)
phpeek-fpm-exporter serve \
  --laravel-site name=App \
  --laravel-site path=/var/www/html \
  --laravel-site queues.redis=default,emails
```

## Configuration

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `PHPEEK_DEBUG` | Enable debug mode | `false` |
| `PHPEEK_MONITOR_LISTEN_ADDR` | Listen address | `:9114` |
| `PHPEEK_PHPFPM_ENABLED` | Enable PHP-FPM monitoring | `true` |
| `PHPEEK_PHPFPM_AUTODISCOVER` | Auto-discover pools | `true` |

### YAML Config

```yaml
debug: false
monitor:
  listen_addr: ":9114"
phpfpm:
  enabled: true
  autodiscover: true
laravel:
  - name: App
    path: /var/www/html
    queues:
      redis: ["default", "emails"]
```

See [Configuration Reference](./docs/configuration.md) for all options.

## Metrics

Key metrics exported:

```text
# PHP-FPM
phpfpm_up{pool="www",socket="..."} 1
phpfpm_active_processes{pool="www"} 2
phpfpm_idle_processes{pool="www"} 3
phpfpm_max_children_reached{pool="www"} 0

# Opcache
phpfpm_opcache_hit_rate{pool="www"} 98.5
phpfpm_opcache_used_memory_bytes{pool="www"} 67108864

# Laravel
laravel_queue_size{site="App",connection="redis",queue="default"} 42
laravel_debug_mode{site="App"} 0
```

See [Metrics Reference](./docs/metrics-reference.md) for the complete list.

## Building

```bash
# Build for current platform
make build

# Build all platforms
make build-all
```

Produces static binaries (no libc dependencies) for:
- `phpeek-fpm-exporter-linux-amd64`
- `phpeek-fpm-exporter-linux-arm64`
- `phpeek-fpm-exporter-darwin-amd64`
- `phpeek-fpm-exporter-darwin-arm64`

## License

MIT License — © 2024–2025 [PHPeek](https://github.com/gophpeek)
