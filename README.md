# Go practices

Go concurrency exercises, an HTTP client example, and a JWT-protected HTTP server.
The repository uses one module, `github.com/jaysonhurd/go-practices`, with Go 1.27.3
or later. Run commands from the repository root.

## Examples

```sh
go run ./concurrency/channels
go run ./concurrency/waitgroups
go run ./concurrency/worker
```

The waitgroups example requires `fixtures/1mb-examplefile-com.txt`, which is not
included; supply that fixture before running it.

Concurrency examples read text under `fixtures/` and print to stdout. Their
structured application logs are appended to `logs/go-practices.log`. These are
learning exercises, not production worker services; the channel example has a
ten-second deadline and deliberately delays output.

For the HTTP client example, supply your own Polygon credential through
`POLYGON_API_KEY`, then run `go run ./httpmusings`. It listens on port 8080 and
proxies the previous-close response for AAPL. The API key is never stored in code.
The client uses a ten-second timeout and releases response bodies. Unit tests
inject `ClientDoer` without reaching Polygon. The existing handler response
behavior is preserved, including an empty response when the upstream call fails.
A credential previously embedded in source should be rotated with its provider;
removing it here does not remove it from Git history.

See [HTTP server usage](httpserver/README.md) for JWT configuration, routes, and
graceful shutdown. It also listens on port 8080, so run the HTTP examples separately.

## Observability and compatibility

All executables use `log/slog` with JSON file output under `logs/`. Generated logs,
local `.codex/` state, IDE files, and JSON files are ignored. No JSON exceptions are
currently required. `go.sum` remains versioned for reproducible dependency checks.

HTTP executables initialize OpenTelemetry tracing and request metrics. Set
`OTEL_STDOUT_ENABLED=true` to export JSON traces and metrics to stdout (otherwise
no telemetry is exported). Exporters are flushed during normal executable cleanup;
both HTTP servers drain requests first. No collector or credentials are needed.
Spans use a fixed name; metrics label only normalized HTTP methods. Request URLs,
headers, and JWT claims are excluded. No global provider is replaced.

OpenTelemetry Go API, SDK, and stdout exporters provide this opt-in
instrumentation. Dependency versions are recorded in `go.mod` and `go.sum`.
Direct and imported transitive dependencies use current releases compatible
with Go 1.27.3. Upstream-only test dependencies in the module graph retain their
upstream requirements; they are not imported by this repository.
Application JSON and test decoding use `encoding/json/v2`, with
`encoding/json/jsontext` for streaming output. The health endpoint retains its
existing JSON body and trailing newline. No experiment flag is required.
Third-party libraries retain control of their own internal JSON implementations.

## Validation

Prerequisites: Go 1.27.3+ (`go` and `gofmt` on PATH), Python 3.9+, Git, and Bash.
There are no external-service requirements or build tags for tests.

```sh
python3 scripts/validate.py
```

This runs Python checker regression tests, repository structure checks, formatting
checks, and `go build ./...`, `go test -race -cover ./...`, and `go vet ./...` in
every discovered nonignored Go module. Formatting failures do not modify files.
For targeted work, use native Go commands from the module root:

```sh
go build ./...
go test -race -cover ./...
go vet ./...
python3 -m unittest discover -s scripts -p 'test_*.py'
python3 scripts/check_repository.py
```

The structural checker covers tracked and new nonignored files. It checks basic
single-line skill frontmatter, inline local Markdown links and heading anchors,
and Bash syntax for `.sh` scripts. It does not check remote links, full YAML,
reference-style Markdown links, or skill behavior. Ignored local `.codex/` skills
are not distributed or checked by CI. Vendored sources are excluded from Go
formatting and module discovery. Tests report coverage; 95% is a goal, not an
artificial pass threshold for executable examples and lifecycle error branches.

GitHub Actions runs the same validation command for pull requests to and pushes
to `main`. Its stable check name is `Repository validation`, with read-only token
permissions, a timeout, SHA-pinned Actions, and no persisted checkout credentials.
Remote branch protections and security settings are not configured by these files.

See [project instructions](AGENTS.md). The original README's personal-preferences
installation guide is preserved as [reference documentation](docs/preferences-reference.md).
