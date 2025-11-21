---
title: "Alerting"
description: "Production-ready Prometheus alerting rules for PHP-FPM, Laravel queues, and Opcache monitoring"
weight: 21
---

# Alerting

Production-ready Prometheus alerting rules for PHPeek metrics.

## Complete Alert Rules

Save as `phpeek-alerts.yml`:

```yaml
groups:
  - name: phpfpm
    interval: 30s
    rules:
      # Critical: PHP-FPM pool is down
      - alert: PHPFPMDown
        expr: phpfpm_up == 0
        for: 1m
        labels:
          severity: critical
        annotations:
          summary: "PHP-FPM pool {{ $labels.pool }} is down"
          description: "Cannot scrape PHP-FPM status from {{ $labels.socket }}"

      # Warning: High process utilization
      - alert: PHPFPMHighUtilization
        expr: phpfpm_active_processes / phpfpm_pm_max_children_config > 0.85
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "PHP-FPM pool {{ $labels.pool }} at {{ $value | humanizePercentage }} capacity"
          description: "Consider increasing pm.max_children or adding more workers"

      # Critical: Process limit reached
      - alert: PHPFPMMaxChildrenReached
        expr: increase(phpfpm_max_children_reached[5m]) > 0
        labels:
          severity: critical
        annotations:
          summary: "PHP-FPM pool {{ $labels.pool }} hit max_children limit"
          description: "Requests are being queued. Increase pm.max_children."

      # Warning: High listen queue
      - alert: PHPFPMListenQueueHigh
        expr: phpfpm_listen_queue > 10
        for: 2m
        labels:
          severity: warning
        annotations:
          summary: "PHP-FPM pool {{ $labels.pool }} has {{ $value }} queued requests"
          description: "Requests are waiting for available workers"

      # Warning: Slow requests increasing
      - alert: PHPFPMSlowRequests
        expr: rate(phpfpm_slow_requests[5m]) > 0.1
        labels:
          severity: warning
        annotations:
          summary: "PHP-FPM pool {{ $labels.pool }} has slow requests"
          description: "Requests exceeding slowlog timeout"

  - name: opcache
    interval: 30s
    rules:
      # Warning: Low hit rate
      - alert: OpcacheLowHitRate
        expr: phpfpm_opcache_hit_rate < 85
        for: 10m
        labels:
          severity: warning
        annotations:
          summary: "Opcache hit rate {{ $value | printf \"%.1f\" }}% on {{ $labels.pool }}"
          description: "Cold cache or undersized opcache settings"

      # Warning: High wasted memory
      - alert: OpcacheHighWaste
        expr: phpfpm_opcache_wasted_memory_percent > 10
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "Opcache {{ $value | printf \"%.1f\" }}% wasted on {{ $labels.pool }}"
          description: "Consider restarting PHP-FPM or tuning validate_timestamps"

      # Critical: Out of memory restarts
      - alert: OpcacheOOMRestarts
        expr: increase(phpfpm_opcache_oom_restarts_total[1h]) > 0
        labels:
          severity: critical
        annotations:
          summary: "Opcache OOM restart on {{ $labels.pool }}"
          description: "Increase opcache.memory_consumption"

      # Warning: Low free memory
      - alert: OpcacheMemoryLow
        expr: phpfpm_opcache_free_memory_bytes < 10000000
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "Opcache <10MB free on {{ $labels.pool }}"
          description: "Cache may start evicting scripts"

  - name: laravel
    interval: 30s
    rules:
      # Warning: Queue backlog
      - alert: LaravelQueueBacklog
        expr: laravel_queue_size > 500
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "Queue backlog {{ $value }} on {{ $labels.site }}/{{ $labels.queue }}"
          description: "Jobs accumulating faster than workers can process"

      # Critical: Large queue backlog
      - alert: LaravelQueueCritical
        expr: laravel_queue_size > 5000
        for: 2m
        labels:
          severity: critical
        annotations:
          summary: "Critical queue backlog {{ $value }} on {{ $labels.site }}/{{ $labels.queue }}"
          description: "Immediate attention required"

      # Critical: Debug mode in production
      - alert: LaravelDebugEnabled
        expr: laravel_debug_mode == 1 and laravel_app_info{env="production"}
        labels:
          severity: critical
        annotations:
          summary: "Debug mode enabled on production {{ $labels.site }}"
          description: "Security risk - disable APP_DEBUG"

      # Warning: Extended maintenance
      - alert: LaravelMaintenanceExtended
        expr: laravel_maintenance_mode == 1
        for: 30m
        labels:
          severity: warning
        annotations:
          summary: "{{ $labels.site }} in maintenance >30 minutes"
          description: "Is maintenance intentional?"

      # Warning: Config not cached
      - alert: LaravelConfigNotCached
        expr: laravel_cache_config == 0 and laravel_app_info{env="production"}
        labels:
          severity: warning
        annotations:
          summary: "Config not cached on {{ $labels.site }}"
          description: "Run 'php artisan config:cache' for better performance"

      # Warning: Routes not cached
      - alert: LaravelRoutesNotCached
        expr: laravel_cache_routes == 0 and laravel_app_info{env="production"}
        labels:
          severity: warning
        annotations:
          summary: "Routes not cached on {{ $labels.site }}"
          description: "Run 'php artisan route:cache' for better performance"
```

## Loading Rules

Add to your Prometheus configuration:

```yaml
# prometheus.yml
rule_files:
  - /etc/prometheus/rules/phpeek-alerts.yml

alerting:
  alertmanagers:
    - static_configs:
        - targets: ['alertmanager:9093']
```

Reload Prometheus:

```bash
curl -X POST http://localhost:9090/-/reload
```

## Severity Levels

| Level | Response | Example |
|-------|----------|---------|
| `critical` | Immediate (page) | FPM down, max_children reached |
| `warning` | Next business day | High utilization, queue backlog |
| `info` | Review weekly | Cache not enabled |

## Alertmanager Routing

Example routing by severity:

```yaml
# alertmanager.yml
route:
  receiver: 'slack-warnings'
  routes:
    - match:
        severity: critical
      receiver: 'pagerduty'
    - match:
        severity: warning
      receiver: 'slack-warnings'

receivers:
  - name: 'pagerduty'
    pagerduty_configs:
      - service_key: '<key>'

  - name: 'slack-warnings'
    slack_configs:
      - api_url: 'https://hooks.slack.com/...'
        channel: '#alerts'
```

## Next Steps

- [Grafana Dashboards](grafana) - Visualize metrics
- [Kubernetes Deployment](kubernetes) - Production deployment
