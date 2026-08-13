# Changelog

All notable changes to this project are documented here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and the project
follows [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Security

- `/json` no longer publishes the full PHP-FPM pool configuration. `php-fpm -tt`
  reports the effective configuration, which idiomatically carries
  `env[DB_PASSWORD]`, `env[APP_KEY]` and `php_admin_value[...]`; the whole map
  was serialised on an unauthenticated endpoint that is enabled by default.
  Only the eleven `pm.*`, `request_*` and `rlimit_*` settings the exporter turns
  into metrics are published now.
- Per-process request URIs are stripped of their query string before they leave
  the process. PHP-FPM reports what each worker last served, so on a real site
  this was a rolling sample of production URLs with their tokens in them.
- Autodiscovery refuses to execute a discovered binary, or read a discovered
  config file, unless it is owned by root or by the exporter and is not writable
  by anyone else. The process table is not a trust boundary: any local user can
  start a process whose name matches and whose command line names a config path
  they control.
- The opcache status script is created with `os.CreateTemp` (random name,
  `O_EXCL`) instead of a fixed path in `/tmp` that was reused if it already
  existed — which let a local user have their own PHP executed as the pool user.
- Queue and connection names are passed to `artisan tinker` as base64-encoded
  JSON instead of being interpolated into the generated PHP.

### Fixed

- **Failures are reported.** Every error in PHP-FPM collection was a `continue`
  plus a log line, so `phpfpm_up` was hardcoded to 1, `phpfpm_scrape_success`
  could never be 0, and a pool that died disappeared from the output instead of
  reporting down. `phpfpm_up == 0` — the alert the metric's help text promises —
  could not fire.
- **Exit codes.** Failing to bind the listen address exited 0, so a port
  collision was a silent crash loop under `Restart=always` and a unit reporting
  SUCCESS under `Restart=on-failure`.
- **Environment variables.** 13 of the 15 documented `CBOX_*` variables did
  nothing: `AutomaticEnv` without a key replacer looked `monitor.listen_addr`
  up as `CBOX_MONITOR.LISTEN_ADDR`. Every documented variable is now covered by
  a test.
- **Manual pool configuration.** Collection dials `status_socket` exclusively
  and only autodiscovery set it, so the documented manual example collected
  nothing. It now falls back to `socket`, and `status_path` defaults to
  `/status`.
- The collector no longer panics when `php artisan about` omits a field.
- Artisan calls are bounded by a context deadline; they previously had none, so
  a hung call kept `/metrics` open indefinitely.
- `--laravel-site` groups keys unambiguously; writing `path=` before `name=`
  used to silently rename the site to `App` and drop the rest of the group.
- Laravel sites merge field by field. Adding `--laravel App:/same/path` on top
  of a config file used to delete every queue it was monitoring.
- App info is cached with a TTL. Failures were cached forever, so an application
  that was down at start-up never reported again.
- PHP version and extensions are cached per binary. On a host running two PHP
  versions, every pool reported whichever was scraped first.
- `version` no longer runs pool autodiscovery: it took a measured 11 seconds and
  printed an error about a subsystem it does not use. Logs go to stderr, so
  `$(fpm-exporter version)` captures only the version.
- Errors are printed once, without a flag dump, and configuration problems are
  reported all at once rather than one per run.
- `make build` is no longer a no-op after the first run (`.PHONY`), stamps the
  version, and `make clean` removes build output instead of a Docker image.

### Added

- `phpfpm_pools_configured`, replacing the synthetic `phpfpm_up{pool="none"}`
  placeholder.
- `/healthz`, which answers without collecting anything, and a landing page at
  `/`. Kubernetes probes should point at `/healthz`; pointing them at
  `/metrics` forks `php artisan` per site on every probe.
- Go runtime, process and `promhttp` handler metrics, plus
  `fpm_exporter_build_info{version,goversion}`.
- `--web.listen-address`, the flag the ecosystem's charts and unit files expect.
- A per-pool `name` setting, used to label a pool that cannot be reached.
- `MIT` LICENSE, `SECURITY.md`, a deterministic CycloneDX SBOM and a dependency
  license gate, all wired into CI.
- Release checksums, verified by `install.sh`.

### Changed

- Pools are scraped concurrently: three unreachable pools cost 3.0s rather than
  a measured 9.0s.
- Default timeouts fit inside Prometheus' own: the scrape budget is 8s (was 15s)
  and the per-site Laravel timeout 5s (was 10s), against a stock
  `scrape_timeout` of 10s.
- `phpfpm_scrape_failures` is a real cumulative counter; it was previously
  emitted with the constant value 1, which made `rate()` meaningless.
- Documentation restructured into the standard topic-folder layout.

### Removed

- `phpfpm.poll_interval` and `php.enabled`, which were documented and read
  nowhere.

## [v3.1.0] - 2026-08-12

First release after the Cbox rebrand. See the
[release notes](https://github.com/cboxdk/fpm-exporter/releases/tag/v3.1.0).

## [v3.0.0] - 2026-01-16

Rebrand from PHPeek to Cbox FPM Exporter, and a redesigned Laravel
configuration with multiple input methods.

[Unreleased]: https://github.com/cboxdk/fpm-exporter/compare/v3.1.0...HEAD
[v3.1.0]: https://github.com/cboxdk/fpm-exporter/releases/tag/v3.1.0
[v3.0.0]: https://github.com/cboxdk/fpm-exporter/releases/tag/v3.0.0
