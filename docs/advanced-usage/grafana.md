---
title: "Grafana Dashboards"
description: "Set up Grafana dashboards for visualizing PHP-FPM, Laravel, and Opcache metrics"
weight: 22
---

# Grafana Dashboards

Visualize PHPeek metrics with Grafana dashboards.

## Quick Start

Import the included dashboard:

1. Open Grafana → Dashboards → Import
2. Upload `testing/grafana/provisioning/dashboards/phpeek-dashboard.json`
3. Select your Prometheus data source
4. Click Import

## Dashboard Panels

### PHP-FPM Overview

**Active/Idle Processes**
```promql
phpfpm_active_processes{pool="$pool"}
phpfpm_idle_processes{pool="$pool"}
```

**Process Utilization**
```promql
phpfpm_active_processes / phpfpm_pm_max_children_config * 100
```

**Listen Queue**
```promql
phpfpm_listen_queue{pool="$pool"}
phpfpm_max_listen_queue{pool="$pool"}
```

### Opcache Health

**Hit Rate Gauge**
```promql
phpfpm_opcache_hit_rate{pool="$pool"}
```

**Memory Usage**
```promql
phpfpm_opcache_used_memory_bytes{pool="$pool"}
phpfpm_opcache_free_memory_bytes{pool="$pool"}
phpfpm_opcache_wasted_memory_bytes{pool="$pool"}
```

### Laravel Queues

**Queue Sizes**
```promql
laravel_queue_size{site="$site"}
```

**Queue Growth Rate**
```promql
rate(laravel_queue_size[5m])
```

## Custom Dashboard JSON

Create a comprehensive dashboard:

```json
{
  "title": "PHPeek PHP-FPM Exporter",
  "templating": {
    "list": [
      {
        "name": "pool",
        "type": "query",
        "query": "label_values(phpfpm_up, pool)"
      },
      {
        "name": "site",
        "type": "query",
        "query": "label_values(laravel_app_info, site)"
      }
    ]
  },
  "panels": [
    {
      "title": "FPM Process Utilization",
      "type": "gauge",
      "targets": [
        {
          "expr": "phpfpm_active_processes{pool=\"$pool\"} / phpfpm_pm_max_children_config{pool=\"$pool\"} * 100"
        }
      ],
      "fieldConfig": {
        "defaults": {
          "thresholds": {
            "steps": [
              {"color": "green", "value": null},
              {"color": "yellow", "value": 70},
              {"color": "red", "value": 90}
            ]
          },
          "unit": "percent",
          "max": 100
        }
      }
    },
    {
      "title": "Opcache Hit Rate",
      "type": "stat",
      "targets": [
        {
          "expr": "phpfpm_opcache_hit_rate{pool=\"$pool\"}"
        }
      ],
      "fieldConfig": {
        "defaults": {
          "unit": "percent",
          "thresholds": {
            "steps": [
              {"color": "red", "value": null},
              {"color": "yellow", "value": 85},
              {"color": "green", "value": 95}
            ]
          }
        }
      }
    },
    {
      "title": "Queue Sizes",
      "type": "timeseries",
      "targets": [
        {
          "expr": "laravel_queue_size{site=\"$site\"}",
          "legendFormat": "{{queue}}"
        }
      ]
    }
  ]
}
```

## Recommended Panels

### PHP-FPM Section
- Process utilization gauge (0-100%)
- Active vs Idle processes (stacked area)
- Listen queue depth (time series)
- Max children reached events (stat)
- Request duration histogram

### Opcache Section
- Hit rate gauge (with thresholds)
- Memory pie chart (used/free/wasted)
- Cached scripts count
- Restart events counter

### Laravel Section
- Queue size by site (bar chart)
- Queue growth rate (time series)
- Application info table
- Debug mode indicator
- Cache status indicators

### System Section
- CPU/Memory limits
- System type info

## Provisioning

Auto-provision dashboards in Grafana:

```yaml
# /etc/grafana/provisioning/dashboards/default.yml
apiVersion: 1
providers:
  - name: 'PHPeek'
    folder: 'PHP Monitoring'
    type: file
    options:
      path: /var/lib/grafana/dashboards/phpeek
```

Place dashboard JSON files in `/var/lib/grafana/dashboards/phpeek/`.

## Variables

Use template variables for multi-pool/site views:

```yaml
# Pool variable
name: pool
type: query
query: label_values(phpfpm_up, pool)
multi: true
includeAll: true

# Site variable
name: site
type: query
query: label_values(laravel_app_info, site)
multi: true
includeAll: true

# Socket variable
name: socket
type: query
query: label_values(phpfpm_up{pool="$pool"}, socket)
```

## Next Steps

- [Kubernetes Deployment](kubernetes) - Deploy in Kubernetes
- [Alerting](alerting) - Set up alerts
