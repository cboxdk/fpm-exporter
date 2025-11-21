---
title: "PHP-FPM Monitoring"
description: "Monitor PHP-FPM pools with automatic discovery, process statistics, and pool configuration metrics"
weight: 11
---

# PHP-FPM Monitoring

PHPeek automatically discovers and monitors PHP-FPM pools via the FastCGI protocol.

## How It Works

1. **Discovery** - Scans running processes for `php-fpm: master process`
2. **Config Parsing** - Runs `php-fpm -tt` to extract pool configurations
3. **Status Collection** - Connects to each pool's status page via FastCGI
4. **Metrics Export** - Exposes Prometheus metrics on `/metrics`

## Prerequisites

Enable the status page in your PHP-FPM pool configuration:

```ini
; /etc/php/8.3/fpm/pool.d/www.conf
[www]
pm.status_path = /status
```

Restart PHP-FPM:

```bash
sudo systemctl restart php-fpm
```

## Automatic Discovery

By default, the exporter discovers pools automatically:

```bash
phpeek-fpm-exporter serve
```

Discovery looks for:
- Running `php-fpm: master process` processes
- Parses config files referenced in command line
- Identifies listen sockets and status paths

## Manual Configuration

For environments where autodiscovery doesn't work:

```yaml
phpfpm:
  autodiscover: false
  pools:
    - socket: "unix:///var/run/php-fpm.sock"
      status_path: /status

    - socket: "tcp://127.0.0.1:9000"
      status_path: /status
```

## Key Metrics

### Pool Health

```promql
# Active process utilization
phpfpm_active_processes / phpfpm_pm_max_children_config * 100

# Listen queue saturation (high = bottleneck)
phpfpm_listen_queue / phpfpm_listen_queue_length * 100

# Process limit breaches
increase(phpfpm_max_children_reached[1h])
```

### Performance

```promql
# Average request duration (last request per process)
avg(phpfpm_process_request_duration) by (pool)

# Slow requests rate
rate(phpfpm_slow_requests[5m])

# Memory per process
avg(phpfpm_process_current_rss) by (pool)
```

## Process Manager Modes

Metrics help tune your PM mode:

### Static (`pm = static`)
- Fixed number of processes
- Watch: `phpfpm_total_processes` should equal `pm.max_children`

### Dynamic (`pm = dynamic`)
- Scales within min/max spare bounds
- Watch: `phpfpm_idle_processes` should stay between min/max spare

### Ondemand (`pm = ondemand`)
- Spawns processes as needed
- Watch: `phpfpm_total_processes` during traffic spikes

## Alerting Examples

```yaml
groups:
  - name: phpfpm
    rules:
      - alert: PHPFPMDown
        expr: phpfpm_up == 0
        for: 1m
        labels:
          severity: critical

      - alert: PHPFPMHighUtilization
        expr: phpfpm_active_processes / phpfpm_pm_max_children_config > 0.9
        for: 5m
        labels:
          severity: warning

      - alert: PHPFPMMaxChildrenReached
        expr: increase(phpfpm_max_children_reached[5m]) > 0
        labels:
          severity: warning

      - alert: PHPFPMHighQueueLength
        expr: phpfpm_listen_queue > 10
        for: 2m
        labels:
          severity: warning
```

## Troubleshooting

### No Pools Discovered

```bash
# Check PHP-FPM is running
pgrep -a php-fpm

# Run with debug mode
PHPEEK_DEBUG=true phpeek-fpm-exporter serve
```

### Permission Denied

```bash
# Check socket permissions
ls -la /var/run/php-fpm.sock

# Add user to www-data group
sudo usermod -a -G www-data $USER
```

### Status Returns Empty

Verify status path is accessible:

```bash
SCRIPT_NAME=/status SCRIPT_FILENAME=/status REQUEST_METHOD=GET \
cgi-fcgi -bind -connect /var/run/php-fpm.sock
```

## Next Steps

- [Laravel Monitoring](laravel-monitoring) - Add Laravel metrics
- [Opcache Metrics](opcache-metrics) - Monitor PHP opcache
