---
title: "Kubernetes Deployment"
description: "Deploy PHPeek PHP-FPM Exporter as a sidecar container or DaemonSet in Kubernetes"
weight: 23
---

# Kubernetes Deployment

Deploy PHPeek PHP-FPM Exporter in Kubernetes environments.

## Sidecar Pattern

Run the exporter alongside your PHP-FPM container:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: php-app
spec:
  replicas: 3
  selector:
    matchLabels:
      app: php-app
  template:
    metadata:
      labels:
        app: php-app
      annotations:
        prometheus.io/scrape: "true"
        prometheus.io/port: "9114"
        prometheus.io/path: "/metrics"
    spec:
      containers:
        # PHP-FPM container
        - name: php-fpm
          image: php:8.3-fpm
          ports:
            - containerPort: 9000
          volumeMounts:
            - name: fpm-socket
              mountPath: /var/run
            - name: app-code
              mountPath: /var/www/html

        # PHPeek Exporter sidecar
        - name: phpeek-exporter
          image: gophpeek/phpeek-fpm-exporter:latest
          args:
            - serve
            - --laravel
            - "name=App,path=/var/www/html"
          ports:
            - containerPort: 9114
              name: metrics
          volumeMounts:
            - name: fpm-socket
              mountPath: /var/run
            - name: app-code
              mountPath: /var/www/html
              readOnly: true
          resources:
            requests:
              cpu: 10m
              memory: 32Mi
            limits:
              cpu: 100m
              memory: 64Mi

      volumes:
        - name: fpm-socket
          emptyDir: {}
        - name: app-code
          # Your application code volume
          persistentVolumeClaim:
            claimName: app-code
```

## ConfigMap Configuration

Use ConfigMap for complex configurations:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: phpeek-config
data:
  config.yaml: |
    debug: false
    logging:
      level: info
      format: json
    monitor:
      listen_addr: ":9114"
    phpfpm:
      autodiscover: true
      poll_interval: 5s
    laravel:
      - name: App
        path: /var/www/html
        queues:
          redis:
            - default
            - emails
---
apiVersion: apps/v1
kind: Deployment
spec:
  template:
    spec:
      containers:
        - name: phpeek-exporter
          image: gophpeek/phpeek-fpm-exporter:latest
          args:
            - serve
            - --config
            - /etc/phpeek/config.yaml
          volumeMounts:
            - name: config
              mountPath: /etc/phpeek
      volumes:
        - name: config
          configMap:
            name: phpeek-config
```

## ServiceMonitor (Prometheus Operator)

If using Prometheus Operator:

```yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: phpeek
  labels:
    release: prometheus
spec:
  selector:
    matchLabels:
      app: php-app
  endpoints:
    - port: metrics
      interval: 30s
      path: /metrics
---
apiVersion: v1
kind: Service
metadata:
  name: php-app-metrics
  labels:
    app: php-app
spec:
  selector:
    app: php-app
  ports:
    - name: metrics
      port: 9114
      targetPort: 9114
```

## PodMonitor Alternative

For direct pod monitoring:

```yaml
apiVersion: monitoring.coreos.com/v1
kind: PodMonitor
metadata:
  name: phpeek
spec:
  selector:
    matchLabels:
      app: php-app
  podMetricsEndpoints:
    - port: metrics
      interval: 30s
```

## Health Probes

Add health checks to the exporter:

```yaml
containers:
  - name: phpeek-exporter
    livenessProbe:
      httpGet:
        path: /metrics
        port: 9114
      initialDelaySeconds: 10
      periodSeconds: 30
    readinessProbe:
      httpGet:
        path: /metrics
        port: 9114
      initialDelaySeconds: 5
      periodSeconds: 10
```

## Resource Recommendations

| Workload | CPU Request | CPU Limit | Memory Request | Memory Limit |
|----------|-------------|-----------|----------------|--------------|
| Light | 10m | 50m | 32Mi | 64Mi |
| Medium | 20m | 100m | 64Mi | 128Mi |
| Heavy | 50m | 200m | 128Mi | 256Mi |

## Security Context

Run with minimal privileges:

```yaml
containers:
  - name: phpeek-exporter
    securityContext:
      runAsNonRoot: true
      runAsUser: 1000
      readOnlyRootFilesystem: true
      allowPrivilegeEscalation: false
      capabilities:
        drop:
          - ALL
```

## Network Policies

Restrict exporter traffic:

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: phpeek-exporter
spec:
  podSelector:
    matchLabels:
      app: php-app
  policyTypes:
    - Ingress
  ingress:
    - from:
        - namespaceSelector:
            matchLabels:
              name: monitoring
      ports:
        - port: 9114
```

## Helm Values

If packaging as Helm chart:

```yaml
# values.yaml
phpeek:
  enabled: true
  image:
    repository: gophpeek/phpeek-fpm-exporter
    tag: latest
  resources:
    requests:
      cpu: 10m
      memory: 32Mi
    limits:
      cpu: 100m
      memory: 64Mi
  config:
    debug: false
    phpfpm:
      autodiscover: true
    laravel: []
  serviceMonitor:
    enabled: true
    interval: 30s
```

## Troubleshooting

### Cannot Connect to FPM Socket

Ensure shared volume is mounted correctly:

```bash
kubectl exec -it <pod> -c phpeek-exporter -- ls -la /var/run/
```

### No Metrics from Laravel

Check app code is accessible:

```bash
kubectl exec -it <pod> -c phpeek-exporter -- ls -la /var/www/html/artisan
```

### High Memory Usage

Reduce poll interval or disable per-process metrics for large pools.

## Next Steps

- [Alerting](alerting) - Production alerts
- [Grafana Dashboards](grafana) - Visualization
