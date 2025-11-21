---
title: "Migration Guide"
description: "Migrate from old Laravel configuration format to new improved syntax"
weight: 30
---

# Migration Guide: Laravel Configuration

**Breaking Change:** Version 2.0 introduces a new, more flexible Laravel configuration format. The old comma-separated format has been removed.

## Why the Change?

The old format had several limitations:

- ❌ Brittle parsing with commas and pipes
- ❌ Poor error messages
- ❌ Difficult to extend
- ❌ Inconsistent with YAML config
- ❌ Hard to discover parameters

The new format provides:

- ✅ Multiple input methods (shorthand, flags, file, env vars)
- ✅ Clear, descriptive errors
- ✅ Easy to extend with new parameters
- ✅ Consistent with YAML configuration
- ✅ Better validation (path existence, artisan file check)

## Migration Examples

### Example 1: Simple Single Site

**Old Format:**
```bash
phpeek-fpm-exporter serve \
  --laravel "name=App,path=/var/www/html"
```

**New Format (Shorthand):**
```bash
phpeek-fpm-exporter serve \
  --laravel App:/var/www/html
```

**New Format (Explicit):**
```bash
phpeek-fpm-exporter serve \
  --laravel-site name=App \
  --laravel-site path=/var/www/html
```

---

### Example 2: Site with Queue Monitoring

**Old Format:**
```bash
phpeek-fpm-exporter serve \
  --laravel "name=App,path=/var/www/html,connection=redis,queues=default|emails"
```

**New Format (Repeatable Flags):**
```bash
phpeek-fpm-exporter serve \
  --laravel-site name=App \
  --laravel-site path=/var/www/html \
  --laravel-site queues.redis=default,emails
```

**New Format (Config File):**

Create `laravel-sites.yaml`:
```yaml
laravel:
  - name: App
    path: /var/www/html
    queues:
      redis:
        - default
        - emails
```

Run:
```bash
phpeek-fpm-exporter serve --laravel-config laravel-sites.yaml
```

---

### Example 3: Multiple Sites

**Old Format:**
```bash
phpeek-fpm-exporter serve \
  --laravel "name=App,path=/var/www/html" \
  --laravel "name=Admin,path=/var/www/admin"
```

**New Format (Repeatable Flags):**
```bash
phpeek-fpm-exporter serve \
  --laravel-site name=App \
  --laravel-site path=/var/www/html \
  --laravel-site name=Admin \
  --laravel-site path=/var/www/admin
```

**New Format (Config File):**

Create `sites.yaml`:
```yaml
laravel:
  - name: App
    path: /var/www/html
  - name: Admin
    path: /var/www/admin
```

Run:
```bash
phpeek-fpm-exporter serve --laravel-config sites.yaml
```

---

### Example 4: Complex Multi-Site with Queues

**Old Format:**
```bash
phpeek-fpm-exporter serve \
  --laravel "name=App,path=/var/www/html,appinfo=true,connection=redis,queues=default|emails" \
  --laravel "name=Admin,path=/var/www/admin,connection=database,queues=jobs"
```

**New Format (Config File - Recommended):**

Create `laravel-sites.yaml`:
```yaml
laravel:
  - name: App
    path: /var/www/html
    enable_app_info: true
    queues:
      redis:
        - default
        - emails

  - name: Admin
    path: /var/www/admin
    queues:
      database:
        - jobs
```

Run:
```bash
phpeek-fpm-exporter serve --laravel-config laravel-sites.yaml
```

---

## New Features

### 1. Shorthand Syntax

For quick, simple setups:

```bash
# Just path (name defaults to "App")
phpeek-fpm-exporter serve --laravel /var/www/html

# Name and path
phpeek-fpm-exporter serve --laravel MyApp:/var/www/html
```

### 2. Environment Variables

**Single Site:**
```bash
export PHPEEK_LARAVEL_SITES='[{"name":"App","path":"/var/www/html","queues":{"redis":["default"]}}]'
phpeek-fpm-exporter serve
```

**Config File Path:**
```bash
export PHPEEK_LARAVEL_CONFIG=/etc/phpeek/laravel-sites.yaml
phpeek-fpm-exporter serve
```

### 3. Config File Reference

Best for complex setups:

```bash
phpeek-fpm-exporter serve --laravel-config /etc/phpeek/sites.yaml
```

### 4. Priority System

Multiple sources can be combined. Priority (highest first):

1. `--laravel-site` flags (CLI)
2. `--laravel` shorthand (CLI)
3. `PHPEEK_LARAVEL_SITES` env var
4. `--laravel-config` file (CLI)
5. `PHPEEK_LARAVEL_CONFIG` file (env)

**Example:**
```bash
# Base config in file
export PHPEEK_LARAVEL_CONFIG=base.yaml

# Override specific site via CLI
phpeek-fpm-exporter serve \
  --laravel-site name=App \
  --laravel-site path=/custom/path
```

This overrides the "App" site from `base.yaml` while keeping others.

---

## Validation Improvements

The new system validates:

- ✅ **Required fields**: `name` and `path` are mandatory
- ✅ **Path existence**: Validates path exists on filesystem
- ✅ **Laravel detection**: Checks for `artisan` file in path
- ✅ **Duplicate names**: Prevents duplicate site names
- ✅ **Clear errors**: Descriptive error messages for all validation failures

**Example Error:**
```
Error: Laravel site 'App' path does not contain artisan file: /var/www/html
```

---

## Quick Conversion Table

| Old Syntax | New Syntax |
|------------|------------|
| `name=App,path=/path` | `--laravel App:/path` or `--laravel-site name=App path=/path` |
| `appinfo=true` | `--laravel-site appinfo=true` |
| `connection=redis,queues=a\|b` | `--laravel-site queues.redis=a,b` |
| Multiple `--laravel` flags | Multiple `--laravel-site` flags or config file |

---

## Docker/Kubernetes Migration

### Old Dockerfile
```dockerfile
CMD ["phpeek-fpm-exporter", "serve", \
     "--laravel", "name=App,path=/var/www/html,connection=redis,queues=default|emails"]
```

### New Dockerfile (Option 1: Shorthand)
```dockerfile
CMD ["phpeek-fpm-exporter", "serve", "--laravel", "/var/www/html"]
```

### New Dockerfile (Option 2: Env Var)
```dockerfile
ENV PHPEEK_LARAVEL_SITES='[{"name":"App","path":"/var/www/html","queues":{"redis":["default","emails"]}}]'
CMD ["phpeek-fpm-exporter", "serve"]
```

### New Dockerfile (Option 3: Config File)
```dockerfile
COPY laravel-sites.yaml /etc/phpeek/
CMD ["phpeek-fpm-exporter", "serve", "--laravel-config", "/etc/phpeek/laravel-sites.yaml"]
```

---

## Troubleshooting

### "Path does not exist" error

**Error:**
```
Laravel site 'App' path does not exist: /var/www/html
```

**Solution:** Ensure the path is correct and exists. The new system validates paths before starting.

### "Path does not contain artisan file" error

**Error:**
```
Laravel site 'App' path does not contain artisan file: /var/www/html
```

**Solution:** Ensure you're pointing to a Laravel application root (the directory containing `artisan` file).

### "Duplicate Laravel site name" error

**Error:**
```
duplicate Laravel site name: App
```

**Solution:** Each site must have a unique name. Use different names for different sites.

### Env var JSON parsing error

**Error:**
```
failed to parse PHPEEK_LARAVEL_SITES: invalid character...
```

**Solution:** Ensure JSON is properly formatted and quoted:
```bash
export PHPEEK_LARAVEL_SITES='[{"name":"App","path":"/var/www/html"}]'
#                             ^                                        ^
#                             Single quotes around JSON
```

---

## Getting Help

- [Configuration Reference](configuration) - Complete configuration guide
- [Quickstart](quickstart) - Quick examples
- [GitHub Issues](https://github.com/gophpeek/phpeek-fpm-exporter/issues) - Report problems
