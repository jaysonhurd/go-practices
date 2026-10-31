# Go practices project instructions

## Project overview and architecture

This repository contains standalone Go learning examples and an HTTP middleware
service. The single module is `github.com/jaysonhurd/go-practices`, supporting Go
1.24.2 or later. Run commands from the repository root.

- `concurrency/channels`, `concurrency/waitgroups`, and `concurrency/worker` are
  separate executables reading local text fixtures.
- `httpmusings` is a Polygon HTTP client example. Its consumer-owned `ClientDoer`
  enables tests without external requests.
- `httpserver` owns startup, configuration, signals, and telemetry shutdown.
  Its internal HTTP package owns routing and server lifecycle. Middleware packages
  own authentication, placeholder authorization, and request logging.
- Root `internal/applog` and `internal/telemetry` supply shared runtime support.
- External integration: Polygon HTTPS API only; no database or local services.
  See [HTTP server documentation](httpserver/README.md) for service behavior.

Keep these example boundaries and existing public function signatures intact.
Avoid abstractions without a concrete consumer. Use small interfaces, composition,
and explicit dependencies. Preserve existing in-progress work.

## Build and development

Prerequisites: Go 1.24.2+, Python 3.9+, Git, and Bash. No build tags are required.

```sh
go build ./...
go test ./...
go vet ./...
python3 scripts/validate.py
```

Run the server with `JWT_SECRET` set via `go run ./httpserver`. Run the other HTTP
example with `POLYGON_API_KEY` set via `go run ./httpmusings`. Both use port 8080;
run only one at a time. Run concurrency examples from the repository root so
`fixtures/` resolves. The waitgroups example requires `fixtures/1mb-examplefile-com.txt`.
No migrations or generated code are needed.

## Formatting and Go conventions

- Use idiomatic, straightforward Go and the local Effective Go reference when
  available. Do not refresh reference sources without an explicit request.
- Separate logical sections with blank lines. Prefer grouped const/var declarations
  at the top of the appropriate scope; keep short error checks local.
- Add declaration comments for exported items and functions, including private
  functions. Add JSON tags to serializable struct fields; mark internal runtime
  dependencies `json:"-"`. Test-only table structs need no serialization contract.
- Prefer `net/http`, `log/slog`, and native `testing`. Keep handler tests using
  `httptest`; use `ClientDoer` for client tests.
- Use structured slog attributes and named constants for fixed messages. Write
  JSON logs to `logs/go-practices.log`; ignore generated logs.
- Define stable local errors as documented package-level `Err...` variables.
  Preserve upstream error identity; wrap with `%w` or join with a sentinel when
  useful. Do not turn dynamic errors into mutable package globals.
- Propagate contexts for I/O and respect cancellation. Validate configuration at
  startup. Keep tokens, query strings with credentials, and claims out of logs
  and telemetry. Never hardcode credentials.
- Preserve JSON v1 and existing wire formats: `encoding/json/v2` is unavailable
  on the minimum Go 1.24.2 toolchain. Do not enable experiments or upgrade Go
  merely to apply a preference.
- OpenTelemetry stdout trace and metric exporters are opt-in through
  `OTEL_STDOUT_ENABLED=true` for HTTP executables. Providers belong to the
  executable lifecycle and must be shut down. Do not invent collector endpoints.
- Format changed Go files. Add meaningful success/error tests; aim for 95%
  coverage where practical and report remaining gaps. Preserve learning examples
  rather than adding tests solely for coverage.

## Compatibility and operations

Preserve health JSON and authentication status/header behavior. Authorization is
currently a documented pass-through stub. JWT validation requires an explicit
algorithm allowlist; do not log credentials or weaken verification. Preserve the
server's signing-key copy, graceful shutdown, and configured request timeouts.
Configuration and route details live in the HTTP server README.

## Repository validation and CI

`python3 scripts/validate.py` runs checker regression tests, skill metadata,
inline local Markdown links/anchors, shell syntax, Go formatting, and each
nonignored Go module's build, race tests with coverage, and vet. CI runs the same
command on PRs and pushes to `main` with stable check `Repository validation`.
The metadata/link checker has limited syntax support; see the root README.

Keep Actions SHA-pinned, checkout credentials disabled, read-only token
permissions, and timeouts. Ignore IDE files, local `.codex/` state, generated logs,
and JSON files except specifically required fixtures; document JSON exceptions.
Preserve licenses and third-party notices. Do not publish, change remote settings,
or initialize Spec Kit as part of routine preference application.
