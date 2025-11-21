---
title: "Quickstart"
description: "Get PHPeek PHP-FPM Exporter running in 5 minutes with automatic PHP-FPM discovery"
weight: 3
---

# Quickstart

Get metrics from your PHP-FPM pools in under 5 minutes.

## Prerequisites

- PHP-FPM running with status page enabled
- Network access to PHP-FPM socket (Unix or TCP)

## Step 1: Enable PHP-FPM Status

Ensure your PHP-FPM pool has the status page enabled. In your pool config (e.g., `/etc/php/8.3/fpm/pool.d/www.conf`):

```ini
pm.status_path = /status
```

Restart PHP-FPM after changes:

```bash
sudo systemctl restart php-fpm
```

## Step 2: Start the Exporter

With automatic discovery (recommended):

```bash
phpeek-fpm-exporter serve
```

The exporter will:
1. Discover running PHP-FPM processes
2. Parse their configurations
3. Start collecting metrics
4. Expose Prometheus endpoint on `:9114`

## Step 3: Verify Metrics

```bash
curl http://localhost:9114/metrics
```

You should see metrics like:

```text
# HELP phpfpm_active_processes The number of active PHP-FPM processes.
# TYPE phpfpm_active_processes gauge
phpfpm_active_processes{pool="www",socket="tcp://127.0.0.1:9000"} 2

# HELP phpfpm_idle_processes The number of idle PHP-FPM processes.
# TYPE phpfpm_idle_processes gauge
phpfpm_idle_processes{pool="www",socket="tcp://127.0.0.1:9000"} 3
```

## Step 4: Configure Prometheus

Add to your `prometheus.yml`:

```yaml
scrape_configs:
  - job_name: 'phpeek'
    static_configs:
      - targets: ['localhost:9114']
```

Reload Prometheus:

```bash
curl -X POST http://localhost:9090/-/reload
```

## Adding Laravel Monitoring

Monitor Laravel queue sizes and application state:

```bash
phpeek-fpm-exporter serve \
  --laravel "name=MyApp,path=/var/www/html,connection=redis,queues=default|emails"
```

This exposes:
- `laravel_queue_size{site="MyApp",connection="redis",queue="default"}`
- `laravel_app_info{site="MyApp",...}`
- `laravel_debug_mode{site="MyApp"}`
- And more...

## Debug Mode

Enable debug logging to troubleshoot issues:

```bash
phpeek-fpm-exporter serve --debug
```

Or via environment variable:

```bash
PHPEEK_DEBUG=true phpeek-fpm-exporter serve
```

## Common Issues

### No PHP-FPM Pools Discovered

1. Check PHP-FPM is running: `pgrep php-fpm`
2. Verify status path is configured in pool config
3. Try with debug mode to see discovery details

### Permission Denied on Socket

Run exporter as the same user as PHP-FPM or add to the `www-data` group:

```bash
sudo usermod -a -G www-data $USER
```

### Metrics Show 0 Values

Ensure `pm.status_path` is set and accessible. Test directly:

```bash
SCRIPT_NAME=/status \
SCRIPT_FILENAME=/status \
REQUEST_METHOD=GET \
cgi-fcgi -bind -connect /var/run/php-fpm.sock
```

## Next Steps

- [Configuration](configuration) - Full configuration reference
- [Metrics Reference](metrics-reference) - All available metrics
- [Laravel Integration](basic-usage/laravel-monitoring) - Detailed Laravel setup
