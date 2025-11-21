---
title: "Metrics Reference"
description: "Complete reference of all Prometheus metrics exported by PHPeek PHP-FPM Exporter"
weight: 5
---

# Metrics Reference

PHPeek PHP-FPM Exporter provides metrics in three categories: PHP-FPM, Laravel, and System.

## PHP-FPM Metrics

### Pool Status

| Metric | Type | Description |
|--------|------|-------------|
| `phpfpm_up` | gauge | Whether scraping was successful (1=yes, 0=no) |
| `phpfpm_start_since` | gauge | Seconds since FPM has started |
| `phpfpm_accepted_connections` | counter | Total accepted connections |
| `phpfpm_listen_queue` | gauge | Requests in pending connection queue |
| `phpfpm_max_listen_queue` | gauge | Max requests in queue since start |
| `phpfpm_listen_queue_length` | gauge | Size of pending connection queue |
| `phpfpm_active_processes` | gauge | Number of active processes |
| `phpfpm_idle_processes` | gauge | Number of idle processes |
| `phpfpm_total_processes` | gauge | Total number of processes |
| `phpfpm_max_active_processes` | gauge | Max active processes since start |
| `phpfpm_max_children_reached` | counter | Times process limit was reached |
| `phpfpm_slow_requests` | counter | Requests exceeding slowlog timeout |
| `phpfpm_memory_peak` | gauge | Peak memory usage of pool |

Labels: `pool`, `socket`

### Process Details

| Metric | Type | Description |
|--------|------|-------------|
| `phpfpm_process_state` | gauge | Process state (Idle=1, Running=1) |
| `phpfpm_process_requests` | counter | Requests served by process |
| `phpfpm_process_request_duration` | gauge | Duration of last request (microseconds) |
| `phpfpm_process_last_request_cpu` | gauge | CPU % of last request |
| `phpfpm_process_last_request_memory` | gauge | Memory of last request (bytes) |
| `phpfpm_process_current_rss` | gauge | Process RSS memory (bytes) |

Labels: `pool`, `socket`, `pid`, `state` (for state metric)

### Pool Configuration

| Metric | Type | Description |
|--------|------|-------------|
| `phpfpm_pm_max_children_config` | gauge | Max child processes |
| `phpfpm_pm_start_servers_config` | gauge | Processes at startup |
| `phpfpm_pm_min_spare_servers_config` | gauge | Min idle processes |
| `phpfpm_pm_max_spare_servers_config` | gauge | Max idle processes |
| `phpfpm_pm_max_requests_config` | gauge | Max requests per process |
| `phpfpm_pm_max_spawn_rate_config` | gauge | Max processes spawned/sec |
| `phpfpm_pm_process_idle_timeout_config` | gauge | Idle timeout (seconds) |
| `phpfpm_request_terminate_timeout_config` | gauge | Request timeout (seconds) |
| `phpfpm_request_slowlog_timeout_config` | gauge | Slowlog timeout (seconds) |
| `phpfpm_rlimit_files_config` | gauge | File descriptor limit |
| `phpfpm_rlimit_core_config` | gauge | Core dump size limit |

Labels: `pool`, `socket`

### Opcache Statistics

| Metric | Type | Description |
|--------|------|-------------|
| `phpfpm_opcache_enabled` | gauge | Whether opcache is enabled |
| `phpfpm_opcache_cached_scripts` | gauge | Number of cached scripts |
| `phpfpm_opcache_hits_total` | counter | Total cache hits |
| `phpfpm_opcache_misses_total` | counter | Total cache misses |
| `phpfpm_opcache_hit_rate` | gauge | Cache hit rate percentage |
| `phpfpm_opcache_used_memory_bytes` | gauge | Used opcache memory |
| `phpfpm_opcache_free_memory_bytes` | gauge | Free opcache memory |
| `phpfpm_opcache_wasted_memory_bytes` | gauge | Wasted opcache memory |
| `phpfpm_opcache_wasted_memory_percent` | gauge | Wasted memory percentage |
| `phpfpm_opcache_oom_restarts_total` | counter | Out-of-memory restarts |
| `phpfpm_opcache_hash_restarts_total` | counter | Hash table restarts |
| `phpfpm_opcache_manual_restarts_total` | counter | Manual restarts |
| `phpfpm_opcache_blacklist_misses_total` | counter | Blacklist misses |

Labels: `pool`, `socket`

## Laravel Metrics

### Application Info

| Metric | Type | Description |
|--------|------|-------------|
| `laravel_app_info` | gauge | Application metadata |
| `laravel_debug_mode` | gauge | Debug mode enabled (1=yes) |
| `laravel_maintenance_mode` | gauge | Maintenance mode (1=yes) |

Labels for `laravel_app_info`: `site`, `version`, `env`, `php_version`, `debug_mode`

### Cache Status

| Metric | Type | Description |
|--------|------|-------------|
| `laravel_cache_config` | gauge | Config cache enabled |
| `laravel_cache_routes` | gauge | Routes cache enabled |
| `laravel_cache_events` | gauge | Events cache enabled |
| `laravel_cache_views` | gauge | Views cache enabled |

Labels: `site`

### Driver Information

| Metric | Type | Description |
|--------|------|-------------|
| `laravel_driver_info` | gauge | Configured driver info |

Labels: `site`, `type` (broadcasting, cache, database, logs, mail, queue, session), `value`

### Queue Metrics

| Metric | Type | Description |
|--------|------|-------------|
| `laravel_queue_size` | gauge | Number of jobs in queue |

Labels: `site`, `connection`, `queue`

## System Metrics

| Metric | Type | Description |
|--------|------|-------------|
| `system_info` | gauge | System information |
| `system_cpu_limit` | gauge | Logical CPU limit |
| `system_memory_limit_mb` | gauge | Memory limit in MB |

Labels for `system_info`: `os`, `arch`, `type` (kubernetes, docker, vm, physical)

## Example Queries

### PHP-FPM Health

```promql
# Pool utilization (active / max_children)
phpfpm_active_processes / phpfpm_pm_max_children_config

# Queue pressure
phpfpm_listen_queue / phpfpm_listen_queue_length

# Max children reached rate
rate(phpfpm_max_children_reached[5m])
```

### Opcache Health

```promql
# Hit rate
phpfpm_opcache_hit_rate

# Memory utilization
phpfpm_opcache_used_memory_bytes /
(phpfpm_opcache_used_memory_bytes + phpfpm_opcache_free_memory_bytes)

# Wasted memory threshold alert
phpfpm_opcache_wasted_memory_percent > 5
```

### Laravel Queues

```promql
# Total jobs across all queues
sum(laravel_queue_size) by (site)

# Jobs per connection
sum(laravel_queue_size) by (connection)

# Alert on queue backlog
laravel_queue_size > 1000
```

## Metric Cardinality

To control metric cardinality:

- Process-level metrics include `pid` label - may be high in dynamic pools
- Consider disabling per-process metrics for very large pools
- Queue metrics scale with `connections * queues * sites`

## Next Steps

- [Alerting](advanced-usage/alerting) - Setting up alerts
- [Grafana Dashboards](advanced-usage/grafana) - Visualization setup
