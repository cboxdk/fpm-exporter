# Security Policy

## Supported versions

The latest major version receives security fixes.

## Reporting a vulnerability

Please **do not** open a public issue. Email [sn@cbox.dk](mailto:sn@cbox.dk)
with a description and, if possible, a proof of concept. You will get a
response within a few business days.

Areas of particular interest for this exporter:

- The metrics endpoints (`/metrics`, `/json`) — exposure, and whether any label
  or field leaks application configuration that should not be scrapeable.
- The generated PHP that the exporter asks PHP-FPM and `artisan tinker` to
  execute, and the temporary files it writes.
- Handling of operator-supplied configuration (paths, queue names, binaries)
  that ends up in a subprocess or in generated code.

## Deployment notes

The exporter is designed to run next to PHP-FPM, not on a public interface:

- `/metrics` and `/json` are **unauthenticated**. Bind to localhost or a private
  interface (`monitor.listen_addr`), and restrict access at the network layer.
  `/json` can be turned off entirely with `monitor.enable_json: false`.
- Collecting Laravel metrics runs `php artisan` **as the user the exporter runs
  as**, inside the configured application directory. That user therefore needs
  no more privilege than reading the app and its queue backends.
- `laravel_app_info` exposes application metadata (framework version, PHP
  version, environment, debug mode) as metric labels. It is opt-in per site
  (`appinfo=true` / `enable_app_info: true`) — leave it off if your metrics
  store is less trusted than the host.
