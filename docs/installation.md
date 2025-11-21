---
title: "Installation"
description: "Download and install PHPeek PHP-FPM Exporter on Linux, macOS, Docker, or Kubernetes"
weight: 2
---

# Installation

PHPeek PHP-FPM Exporter is distributed as a single static binary with no runtime dependencies. It works on any Linux distribution including Alpine.

## Quick Install

```bash
# Latest version
curl -fsSL https://raw.githubusercontent.com/gophpeek/phpeek-fpm-exporter/main/install.sh | sh

# Specific version
curl -fsSL https://raw.githubusercontent.com/gophpeek/phpeek-fpm-exporter/main/install.sh | VERSION=v1.0.0 sh

# Custom install directory
curl -fsSL https://raw.githubusercontent.com/gophpeek/phpeek-fpm-exporter/main/install.sh | INSTALL_DIR=/opt/bin sh
```

This auto-detects your OS and architecture and installs to `/usr/local/bin`.

## Pre-built Binaries

Download the latest release from [GitHub Releases](https://github.com/gophpeek/phpeek-fpm-exporter/releases):

```bash
# Linux (amd64) - Works on ALL distributions including Alpine
wget https://github.com/gophpeek/phpeek-fpm-exporter/releases/latest/download/phpeek-fpm-exporter-linux-amd64
chmod +x phpeek-fpm-exporter-linux-amd64
sudo mv phpeek-fpm-exporter-linux-amd64 /usr/local/bin/phpeek-fpm-exporter

# Linux (arm64)
wget https://github.com/gophpeek/phpeek-fpm-exporter/releases/latest/download/phpeek-fpm-exporter-linux-arm64
chmod +x phpeek-fpm-exporter-linux-arm64
sudo mv phpeek-fpm-exporter-linux-arm64 /usr/local/bin/phpeek-fpm-exporter

# macOS (Apple Silicon)
wget https://github.com/gophpeek/phpeek-fpm-exporter/releases/latest/download/phpeek-fpm-exporter-darwin-arm64
chmod +x phpeek-fpm-exporter-darwin-arm64
sudo mv phpeek-fpm-exporter-darwin-arm64 /usr/local/bin/phpeek-fpm-exporter

# macOS (Intel)
wget https://github.com/gophpeek/phpeek-fpm-exporter/releases/latest/download/phpeek-fpm-exporter-darwin-amd64
chmod +x phpeek-fpm-exporter-darwin-amd64
sudo mv phpeek-fpm-exporter-darwin-amd64 /usr/local/bin/phpeek-fpm-exporter
```

## Build from Source

Requires Go 1.24 or later:

```bash
git clone https://github.com/gophpeek/phpeek-fpm-exporter.git
cd phpeek-fpm-exporter

# Build for current platform
make build

# Build for all platforms
make build-all
```

Built binaries are placed in `build/`:
- `build/phpeek-fpm-exporter-linux-amd64`
- `build/phpeek-fpm-exporter-linux-arm64`
- `build/phpeek-fpm-exporter-darwin-amd64`
- `build/phpeek-fpm-exporter-darwin-arm64`

## Docker

Run alongside your PHP-FPM container:

```bash
docker run -d \
  --name phpeek-exporter \
  -p 9114:9114 \
  -v /var/run/php-fpm.sock:/var/run/php-fpm.sock \
  gophpeek/phpeek-fpm-exporter:latest serve
```

## Kubernetes

Deploy as a sidecar container in your PHP-FPM pod:

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: php-app
  annotations:
    prometheus.io/scrape: "true"
    prometheus.io/port: "9114"
spec:
  containers:
    - name: php-fpm
      image: php:8.3-fpm
      volumeMounts:
        - name: fpm-socket
          mountPath: /var/run

    - name: phpeek-exporter
      image: gophpeek/phpeek-fpm-exporter:latest
      args: ["serve"]
      ports:
        - containerPort: 9114
      volumeMounts:
        - name: fpm-socket
          mountPath: /var/run

  volumes:
    - name: fpm-socket
      emptyDir: {}
```

## Systemd Service

Create `/etc/systemd/system/phpeek-fpm-exporter.service`:

```ini
[Unit]
Description=PHPeek PHP-FPM Exporter
After=network.target php-fpm.service

[Service]
Type=simple
User=www-data
ExecStart=/usr/local/bin/phpeek-fpm-exporter serve
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
```

Enable and start:

```bash
sudo systemctl daemon-reload
sudo systemctl enable phpeek-fpm-exporter
sudo systemctl start phpeek-fpm-exporter
```

## Verify Installation

Check the exporter is running:

```bash
# Check version
phpeek-fpm-exporter version

# Start and test metrics endpoint
phpeek-fpm-exporter serve &
curl http://localhost:9114/metrics
```

## Next Steps

- [Quickstart](quickstart) - Configure and run your first scrape
- [Configuration](configuration) - Customize for your environment
