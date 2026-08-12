---
title: "Installation"
description: "Download and install Cbox FPM Exporter on Linux, macOS, Docker, or Kubernetes"
weight: 1
---

# Installation

Cbox FPM Exporter is distributed as a single static binary with no runtime dependencies. It works on any Linux distribution including Alpine.

## Quick Install

```bash
# Latest version
curl -fsSL https://raw.githubusercontent.com/cboxdk/fpm-exporter/main/install.sh | sh

# Specific version
curl -fsSL https://raw.githubusercontent.com/cboxdk/fpm-exporter/main/install.sh | VERSION=v1.0.0 sh

# Custom install directory
curl -fsSL https://raw.githubusercontent.com/cboxdk/fpm-exporter/main/install.sh | INSTALL_DIR=/opt/bin sh
```

This auto-detects your OS and architecture and installs to `/usr/local/bin`.

## Pre-built Binaries

Download the latest release from [GitHub Releases](https://github.com/cboxdk/fpm-exporter/releases):

```bash
# Linux (amd64) - Works on ALL distributions including Alpine
wget https://github.com/cboxdk/fpm-exporter/releases/latest/download/fpm-exporter-linux-amd64
chmod +x fpm-exporter-linux-amd64
sudo mv fpm-exporter-linux-amd64 /usr/local/bin/fpm-exporter

# Linux (arm64)
wget https://github.com/cboxdk/fpm-exporter/releases/latest/download/fpm-exporter-linux-arm64
chmod +x fpm-exporter-linux-arm64
sudo mv fpm-exporter-linux-arm64 /usr/local/bin/fpm-exporter

# macOS (Apple Silicon)
wget https://github.com/cboxdk/fpm-exporter/releases/latest/download/fpm-exporter-darwin-arm64
chmod +x fpm-exporter-darwin-arm64
sudo mv fpm-exporter-darwin-arm64 /usr/local/bin/fpm-exporter

# macOS (Intel)
wget https://github.com/cboxdk/fpm-exporter/releases/latest/download/fpm-exporter-darwin-amd64
chmod +x fpm-exporter-darwin-amd64
sudo mv fpm-exporter-darwin-amd64 /usr/local/bin/fpm-exporter
```

## Build from Source

Requires Go 1.24 or later:

```bash
git clone https://github.com/cboxdk/fpm-exporter.git
cd fpm-exporter

# Build for current platform
make build

# Build for all platforms
make build-all
```

Built binaries are placed in `build/`:
- `build/fpm-exporter-linux-amd64`
- `build/fpm-exporter-linux-arm64`
- `build/fpm-exporter-darwin-amd64`
- `build/fpm-exporter-darwin-arm64`

## Containers

There is no published container image yet. To run the exporter in a container,
copy a release binary into your own image:

```dockerfile
FROM alpine:3.21
COPY fpm-exporter-linux-amd64 /usr/local/bin/fpm-exporter
ENTRYPOINT ["/usr/local/bin/fpm-exporter", "serve"]
```

The binary is static (`CGO_ENABLED=0`), so a scratch or distroless base works
too. See [Kubernetes Deployment](../advanced-usage/kubernetes) for running it as
a sidecar next to PHP-FPM.

## Systemd Service

Create `/etc/systemd/system/fpm-exporter.service`:

```ini
[Unit]
Description=Cbox FPM Exporter
After=network.target php-fpm.service

[Service]
Type=simple
User=www-data
ExecStart=/usr/local/bin/fpm-exporter serve
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
```

Enable and start:

```bash
sudo systemctl daemon-reload
sudo systemctl enable fpm-exporter
sudo systemctl start fpm-exporter
```

## Verify Installation

Check the exporter is running:

```bash
# Check version
fpm-exporter version

# Start and test metrics endpoint
fpm-exporter serve &
curl http://localhost:9114/metrics
```

## Next Steps

- [Quickstart](../quickstart) - Configure and run your first scrape
- [Configuration](../configuration/reference) - Customize for your environment
