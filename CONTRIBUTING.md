# Contributing

Thanks for looking at this.

## Before you open a pull request

Run the same gate CI runs:

```bash
make check
```

That is gofmt, `go vet`, golangci-lint, `go test -race -cover`, govulncheck, an
SBOM drift check and a dependency license check. CI runs the identical set plus
a build of all four release targets, so a green `make check` should mean a green
pull request.

## What the tests are for

Parsing code is tested against **real captured output**, not hand-written
approximations — see `internal/phpfpm/testdata/`. If you change how a PHP-FPM
status payload or a `php-fpm -tt` report is decoded, add a fixture from a real
PHP-FPM rather than a synthetic string. The JSON tags in particular are
space-separated names like `"accepted conn"`, and a plausible-looking tidy-up
silently reports zero forever.

Tests must pass under `go test -shuffle=on -count=2 ./...`. If a test depends on
another test having run first, it is not testing what it says it is.

## Commits

Conventional-commit subjects (`fix(serve): ...`), and a body explaining *why*
when the reasoning is not obvious from the diff. If a change fixes something
that was silently wrong in production, say what the symptom was — the commit log
is where the next person finds out that a metric used to lie.

## Metric changes

Renaming or rescaling a published metric breaks every dashboard and alert built
on it. Add the corrected metric alongside the old one and mark the old one
deprecated, unless the change is going into a major release.
