---
title: "Laravel Monitoring"
description: "Monitor Laravel applications including queue sizes, application info, cache state, and driver configuration"
weight: 12
---

# Laravel Monitoring

Monitor your Laravel applications' health, queue backlogs, and configuration state.

## How It Works

The exporter collects Laravel metrics by executing Artisan commands:

- `php artisan about --json` - Application info, cache state, drivers
- `php artisan tinker --execute` - Queue sizes via Redis/Database queries

## Basic Setup

### CLI

```bash
phpeek-fpm-exporter serve \
  --laravel "name=MyApp,path=/var/www/html"
```

### YAML Config

```yaml
laravel:
  - name: MyApp
    path: /var/www/html
```

## Queue Monitoring

Track queue sizes across connections and queues:

### Single Connection

```bash
--laravel "name=App,path=/var/www/html,connection=redis,queues=default|emails|notifications"
```

### Multiple Connections

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
      sqs:
        - webhooks
```

### Metrics Produced

```text
laravel_queue_size{site="MyApp",connection="redis",queue="default"} 42
laravel_queue_size{site="MyApp",connection="redis",queue="emails"} 156
laravel_queue_size{site="MyApp",connection="database",queue="notifications"} 0
```

## Application Info

Exposes application metadata from `php artisan about`:

```text
laravel_app_info{site="MyApp",version="11.0.0",env="production",php_version="8.3.14",debug_mode="false"} 1
laravel_debug_mode{site="MyApp"} 0
laravel_maintenance_mode{site="MyApp"} 0
```

## Cache State

Track which caches are active:

```text
laravel_cache_config{site="MyApp"} 1    # Config cached
laravel_cache_routes{site="MyApp"} 1    # Routes cached
laravel_cache_events{site="MyApp"} 0    # Events not cached
laravel_cache_views{site="MyApp"} 1     # Views cached
```

## Driver Configuration

Monitor configured drivers:

```text
laravel_driver_info{site="MyApp",type="cache",value="redis"} 1
laravel_driver_info{site="MyApp",type="queue",value="redis"} 1
laravel_driver_info{site="MyApp",type="session",value="database"} 1
laravel_driver_info{site="MyApp",type="database",value="mysql"} 1
```

## Multi-Site Monitoring

Monitor multiple Laravel applications:

```yaml
laravel:
  - name: API
    path: /var/www/api
    queues:
      redis: [default, webhooks]

  - name: Admin
    path: /var/www/admin
    queues:
      database: [reports]

  - name: Worker
    path: /var/www/worker
    queues:
      redis: [jobs, emails, notifications]
```

## Custom PHP Binary

Use a specific PHP version per site:

```yaml
laravel:
  - name: LegacyApp
    path: /var/www/legacy
    php_config:
      binary: /usr/bin/php7.4
    queues:
      database: [jobs]
```

## Alerting Examples

```yaml
groups:
  - name: laravel
    rules:
      - alert: LaravelQueueBacklog
        expr: laravel_queue_size > 1000
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "Queue backlog on {{ $labels.site }}/{{ $labels.queue }}"

      - alert: LaravelDebugEnabled
        expr: laravel_debug_mode == 1
        labels:
          severity: critical
        annotations:
          summary: "Debug mode enabled on {{ $labels.site }}"

      - alert: LaravelMaintenanceMode
        expr: laravel_maintenance_mode == 1
        for: 30m
        labels:
          severity: warning
        annotations:
          summary: "{{ $labels.site }} in maintenance for >30m"

      - alert: LaravelConfigNotCached
        expr: laravel_cache_config == 0
        labels:
          severity: warning
        annotations:
          summary: "Config not cached on {{ $labels.site }}"
```

## PromQL Examples

```promql
# Total jobs across all queues
sum(laravel_queue_size)

# Jobs by site
sum(laravel_queue_size) by (site)

# Jobs by connection type
sum(laravel_queue_size) by (connection)

# Rate of queue growth
rate(laravel_queue_size[5m])

# Sites with debug enabled
count(laravel_debug_mode == 1)
```

## Troubleshooting

### Queue Size Shows 0

1. Verify queue connection is configured
2. Check Redis/database is accessible
3. Test manually: `php artisan tinker --execute="Queue::size('default')"`

### App Info Missing

1. Check `php artisan about --json` works
2. Verify PHP binary path is correct
3. Check Laravel version supports `--json` flag

### Permission Issues

The exporter needs to:
- Read Laravel application files
- Execute `php artisan` commands
- Access queue storage (Redis/Database)

Run as same user as your web server or ensure proper permissions.

## Next Steps

- [Opcache Metrics](opcache-metrics) - Monitor PHP opcache
- [Alerting](../advanced-usage/alerting) - Set up production alerts
