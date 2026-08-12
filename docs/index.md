---
title: "Introduction"
description: "Cbox FPM Exporter - Lightweight Prometheus exporter for PHP-FPM and Laravel applications"
weight: 1
---

# Cbox FPM Exporter

Cbox FPM Exporter is a lightweight, Go-based Prometheus exporter for PHP-FPM and Laravel applications. It provides comprehensive metrics for monitoring PHP-FPM pools, Opcache statistics, and Laravel application state.

## Key Features

- **PHP-FPM Metrics** - Pool status, process counts, request statistics via FastCGI
- **Automatic Discovery** - Discovers PHP-FPM pools automatically using `php-fpm -tt`
- **Opcache Statistics** - Detailed Opcache metrics per FPM pool
- **Laravel Integration** - Queue sizes, application info, cache state, driver configuration
- **Multi-Site Support** - Monitor multiple Laravel applications simultaneously
- **Prometheus Native** - Standard `/metrics` endpoint with proper metric types
- **Zero Dependencies** - Single static binary, works on any Linux distribution

## Use Cases

### Production Monitoring
Monitor PHP-FPM process health, connection queues, and resource usage in real-time. Set up alerts for max_children reached, high queue lengths, or Opcache issues.

### Laravel Queue Monitoring
Track queue sizes across multiple connections (Redis, database, SQS) and queues. Monitor for queue backlogs before they become problems.

### Performance Optimization
Use Opcache metrics to tune memory allocation and identify cache invalidation issues. Monitor hit rates and memory waste percentages.

## Architecture

The exporter runs as a sidecar or standalone service, collecting metrics via:

1. **FastCGI Protocol** - Communicates directly with PHP-FPM sockets for status
2. **PHP CLI** - Executes `php artisan` commands for Laravel metrics
3. **Process Discovery** - Parses running processes to find FPM pools

```
┌─────────────────┐     ┌──────────────────┐     ┌─────────────┐
│   Prometheus    │────▶│  Cbox Exporter   │────▶│   PHP-FPM   │
│                 │     │    :9114         │     │   Pools     │
└─────────────────┘     └──────────────────┘     └─────────────┘
                               │
                               ▼
                        ┌─────────────┐
                        │   Laravel   │
                        │    Apps     │
                        └─────────────┘
```

## Documentation

- [Quickstart](quickstart) - Get running in 5 minutes
- [Requirements](requirements) - What the exporter needs on the host
- [Getting Started](getting-started/) - Installation and migration from older config formats
- [Basic Usage](basic-usage/) - PHP-FPM, Laravel and Opcache monitoring
- [Configuration](configuration/) - Every flag, environment variable and YAML key
- [Advanced Usage](advanced-usage/) - Alerting, Grafana and Kubernetes
- [Reference](reference/) - Complete metrics reference
