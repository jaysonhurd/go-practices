# HTTP server

Requires Go 1.27.3 or later. From the repository root, set a signing secret
and start the server:

```sh
export JWT_SECRET='replace-with-a-long-random-secret'
go run ./httpserver
```

The server listens on `:8080`. Logs are appended as JSON to
`logs/go-practices.log` relative to the working directory. The directory and file
are created at startup; failures stop startup. SIGINT and SIGTERM gracefully
shut down the server and close the log file.

- `GET /health` returns `{"health":"true"}` without authentication.
- `GET /items` returns an empty `200 OK` for a valid HS256 JWT signed with
  `JWT_SECRET`. Send it as `Authorization: Bearer <token>`.
- Missing, malformed, expired, or incorrectly signed tokens return `401`.
- Authorization currently permits every authenticated request and is a stub.

The authentication package accepts a key resolver, algorithm allowlist, and JWT
parser options so other signing algorithms, issuer and audience requirements can
be configured. All verified claims are available through
`authentication.ClaimsFromContext(r.Context())`. Expiration and not-before are
validated when present; expiration is not required by default.

```sh
go build ./httpserver/...
go test -race ./httpserver/...
go vet ./httpserver/...
```

Run commands from the repository root so the log file is written to the root
`logs/` directory. Server setup requires a non-nil logger and a nonempty signing
secret. The server retains its own copy of the signing key.

The executable owns signal handling and passes a cancellation context to
`Server.Run(ctx)`. Cancellation drains active requests for up to ten seconds
before forcing connections closed. An already-canceled context prevents startup.
The HTTP server uses a five-second header timeout, fifteen-second read and write
timeouts, and a sixty-second idle timeout.

`internal/http` owns routes and the server lifecycle. Middleware packages own JWT
authentication, the authorization placeholder, and structured request logging.
Their unit tests live alongside the corresponding implementations; route and
lifecycle integration tests live in `internal/http`.

Set `OTEL_STDOUT_ENABLED=true` before startup to export OpenTelemetry HTTP spans,
request counts, and duration metrics to stdout. Export is disabled by default.
The executable owns both providers and flushes them after HTTP shutdown with a
five-second timeout. See the [repository documentation](../README.md#observability-and-compatibility)
for dependency and privacy details.
