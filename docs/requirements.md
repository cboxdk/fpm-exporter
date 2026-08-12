---
title: "Requirements"
description: "Platforms, PHP and Laravel versions Cbox FPM Exporter needs on the host"
weight: 2
---

# Requirements

## Running the exporter

The exporter ships as a single static binary with no runtime dependencies.

| | |
|---|---|
| **Platforms** | `linux/amd64`, `linux/arm64`, `darwin/amd64`, `darwin/arm64` — the four targets built by the release workflow |
| **Privileges** | Read access to the PHP-FPM status socket, and to the pool config files when autodiscovery is on |
| **Network** | One listening port for the metrics endpoint (`:9114` by default) |

## PHP-FPM metrics

| | |
|---|---|
| **PHP-FPM** | Any version that serves a status page over FastCGI |
| **Status page** | Enabled per pool (`pm.status_path`), or reachable on a separate status socket |
| **Autodiscovery** | Requires the `php-fpm` binary on `PATH`, since discovery runs `php-fpm -tt` to read the effective pool configuration. Disable with `--autodiscover=false` and configure pools manually |
| **Opcache metrics** | The `opcache` extension loaded in the FPM pool |

## Laravel metrics

Laravel collection is opt-in per site and runs Artisan on the host, so it needs
more than the FastCGI socket:

| | |
|---|---|
| **PHP CLI** | A `php` binary the exporter can execute (`php.binary`, or `php_config.binary` per site) |
| **Application** | A readable application root containing `artisan` — the exporter refuses to start otherwise |
| **Queue metrics** | `php artisan tinker`, which means `laravel/tinker` installed in the application |
| **App info** | `php artisan about --json`, available from Laravel 9 onwards |
| **Permissions** | The exporter's user must be able to boot the application and reach its queue backends |

Each Artisan call is bounded by the site's `timeout` (10s by default) and by the
overall `monitor.scrape_timeout` (15s by default).

## Building from source

| | |
|---|---|
| **Go** | 1.25 or newer (`go.mod`); CI builds and tests on 1.26 |
| **Toolchain** | `CGO_ENABLED=0`; no C compiler needed |
