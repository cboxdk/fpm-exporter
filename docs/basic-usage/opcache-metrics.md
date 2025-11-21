---
title: "Opcache Metrics"
description: "Monitor PHP Opcache statistics per FPM pool including hit rates, memory usage, and cache invalidations"
weight: 13
---

# Opcache Metrics

PHPeek collects detailed Opcache statistics for each PHP-FPM pool, helping you optimize PHP performance.

## How It Works

The exporter:
1. Sends a PHP script via FastCGI to each pool
2. Script executes `opcache_get_status()` and returns JSON
3. Metrics are parsed and exposed to Prometheus

This approach gives you per-pool Opcache visibility, unlike system-wide tools.

## Key Metrics

### Cache Efficiency

| Metric | Description |
|--------|-------------|
| `phpfpm_opcache_hit_rate` | Percentage of cache hits |
| `phpfpm_opcache_hits_total` | Total cache hits |
| `phpfpm_opcache_misses_total` | Total cache misses |
| `phpfpm_opcache_cached_scripts` | Number of cached scripts |

### Memory Usage

| Metric | Description |
|--------|-------------|
| `phpfpm_opcache_used_memory_bytes` | Used memory |
| `phpfpm_opcache_free_memory_bytes` | Free memory |
| `phpfpm_opcache_wasted_memory_bytes` | Wasted memory |
| `phpfpm_opcache_wasted_memory_percent` | Waste percentage |

### Restarts

| Metric | Description |
|--------|-------------|
| `phpfpm_opcache_oom_restarts_total` | Out-of-memory restarts |
| `phpfpm_opcache_hash_restarts_total` | Hash table full restarts |
| `phpfpm_opcache_manual_restarts_total` | Manual restarts |

## PromQL Examples

### Cache Health

```promql
# Hit rate (should be >95%)
phpfpm_opcache_hit_rate

# Memory utilization
phpfpm_opcache_used_memory_bytes /
(phpfpm_opcache_used_memory_bytes + phpfpm_opcache_free_memory_bytes) * 100

# Wasted memory (should be <5%)
phpfpm_opcache_wasted_memory_percent
```

### Problem Detection

```promql
# Low hit rate indicates cold cache or undersized
phpfpm_opcache_hit_rate < 90

# High waste indicates need for opcache.max_wasted_percentage tune
phpfpm_opcache_wasted_memory_percent > 5

# OOM restarts indicate undersized opcache.memory_consumption
increase(phpfpm_opcache_oom_restarts_total[1h]) > 0

# Hash restarts indicate undersized opcache.max_accelerated_files
increase(phpfpm_opcache_hash_restarts_total[1h]) > 0
```

## Alerting Examples

```yaml
groups:
  - name: opcache
    rules:
      - alert: OpcacheLowHitRate
        expr: phpfpm_opcache_hit_rate < 90
        for: 10m
        labels:
          severity: warning
        annotations:
          summary: "Opcache hit rate {{ $value }}% on {{ $labels.pool }}"

      - alert: OpcacheHighWaste
        expr: phpfpm_opcache_wasted_memory_percent > 10
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "Opcache {{ $value }}% wasted on {{ $labels.pool }}"

      - alert: OpcacheOOMRestarts
        expr: increase(phpfpm_opcache_oom_restarts_total[1h]) > 0
        labels:
          severity: critical
        annotations:
          summary: "Opcache OOM restart on {{ $labels.pool }}"

      - alert: OpcacheMemoryFull
        expr: phpfpm_opcache_free_memory_bytes < 10000000
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "Opcache < 10MB free on {{ $labels.pool }}"
```

## Tuning Recommendations

### Low Hit Rate

**Symptom**: `phpfpm_opcache_hit_rate < 90`

**Causes**:
- Cache cold after restart
- `opcache.validate_timestamps=1` with frequent deployments
- Undersized `opcache.max_accelerated_files`

**Solutions**:
- Preload cache: `opcache.preload`
- Disable timestamps in production: `opcache.validate_timestamps=0`
- Increase max files: `opcache.max_accelerated_files=20000`

### High Wasted Memory

**Symptom**: `phpfpm_opcache_wasted_memory_percent > 5`

**Causes**:
- Frequent code changes without restart
- `opcache.validate_timestamps=1` in production

**Solutions**:
- Restart PHP-FPM after deployments
- Adjust `opcache.max_wasted_percentage`

### OOM Restarts

**Symptom**: `phpfpm_opcache_oom_restarts_total` increasing

**Causes**:
- Undersized `opcache.memory_consumption`
- Large codebase

**Solutions**:
- Increase memory: `opcache.memory_consumption=256`
- Use preloading to reduce overhead

### Hash Restarts

**Symptom**: `phpfpm_opcache_hash_restarts_total` increasing

**Causes**:
- More scripts than `opcache.max_accelerated_files`

**Solutions**:
- Increase: `opcache.max_accelerated_files=30000`
- Note: Must be prime number for optimal hashing

## php.ini Recommendations

Production settings:

```ini
opcache.enable=1
opcache.memory_consumption=256
opcache.interned_strings_buffer=16
opcache.max_accelerated_files=20000
opcache.validate_timestamps=0
opcache.save_comments=1
opcache.enable_cli=0
```

Development settings:

```ini
opcache.enable=1
opcache.memory_consumption=128
opcache.validate_timestamps=1
opcache.revalidate_freq=0
```

## Next Steps

- [Alerting](../advanced-usage/alerting) - Production alert setup
- [Grafana Dashboards](../advanced-usage/grafana) - Visualize Opcache trends
