# lunar-gateway — c12s HTTP API gateway

The single HTTP entry point to the control plane. Every `cockpit` command and
every operator HTTP call lands here; the gateway translates it into a gRPC
call to the owning service (Magnetar, Gravity, Kuiper, Quasar, Apollo, Oort,
Meridian, Supernova) using **server reflection** — there is no hand-written
per-method proxying code. Routes are declared in config.

## Role in c12s

```
cockpit / curl ──HTTP──▶ lunar-gateway (:5555) ──gRPC──▶ Magnetar, Gravity,
                          │                              Kuiper, Quasar, ...
                          ├─ VerifyToken ──▶ Apollo (JWT)
                          └─ allow? ──────▶ rate_limiter_service (heliosphere)
```

- `config/config.yml` declares every route: `(group, version, action name)`
  → `method_route` + HTTP `type` + target `service`. Cockpit reads this same
  file (`CONFIG_PATH=../lunar-gateway/config/config.yml`) to build its URLs.
- `config/no_auth_config.yml` lists the methods that need no token
  (registration, login, node pool listing, rate-limiter management).
- Response/request bodies are JSON or protobuf, converted through the
  reflected service descriptors (grpcurl descriptor source).

## Authentication and authorization

1. `Authorization: Bearer <token>` is required for all non-no-auth routes.
   The `Bearer ` prefix is stripped before use (JWT itself). A missing or
   invalid token → `401`.
2. The token is verified with Apollo (`VerifyToken`); the returned JWT is
   forwarded to the target service as gRPC metadata `authz-token`.
3. Services (Magnetar, Gravity, Kuiper, Supernova) enforce their own
   `permissions` claim checks — the gateway is not the authorization point.
   Supernova checks `image.build|image|<repo path>`; the other services check
   their node/config permissions the same way.

## Rate limiting

Every **authenticated** request consults `rate_limiter_service`
(heliosphere, Consul-backed) before being forwarded, keyed
`<methodName>-<username>`. If no limiter is configured for that key, a
**default is auto-created: token bucket, burst 1, 1 request/second**. Scripts
that fire several requests at one endpoint in quick succession will get
`User reached rate limit per request.` — space calls, or create a custom
limiter for the user/method (`/rate-limiter` no-auth routes).

A second, system-wide limiter can be enabled by starting the binary with the
`sysrl` argument (disabled in the compose deployment; see `Dockerfile`).

## Configuration

| File | Purpose |
|---|---|
| `config/config.yml` | services (host:port) + route table (51 routes today) |
| `config/no_auth_config.yml` | routes exempt from authentication |
| `.env` (compose) | gateway port for the container |
| `GATEWAY_STARTUP_TIMEOUT` | how long startup retries each backend (default `5m`) |

## Startup and reboot behavior

At startup the gateway dials every service in `config/config.yml`, reflects
its gRPC services, and builds the route handlers. It does **not** assume the
control plane is already up: each backend is retried every 3 s until reachable,
and startup fails cleanly after `GATEWAY_STARTUP_TIMEOUT` (default 5 minutes)
if a backend never appears. This matters after a host or Docker daemon restart,
because Docker starts containers in arbitrary order and Compose's
`depends_on` conditions only apply to an initial `compose up`.

Previously a single unavailable backend caused `panic: context deadline
exceeded` and a gateway crash loop. The retry loop replaces that behavior;
`docker-compose.yml` now also uses `restart: unless-stopped` for the gateway.

## Build, run

```bash
go build ./...       # no tests exist (see bugs)
go run .             # needs config.yml + no_auth_config.yml in CWD
                     # and the target services reachable at compose hostnames
```

The Dockerfile builds from the **workspace root** context (it copies
`oort/`, `magnetar/`, `apollo/`, `heliosphere/` next to the gateway) and runs
`./main` on port 5555. In compose it is the `lunar-gateway` container; the
lab exposes `localhost:5555`.

## Status

Stable and load-bearing: every cockpit call and all operator HTTP traffic
passes through it. The route table is hand-maintained; only routes present
in `config/config.yml` are reachable.

### Build routes (Supernova)

`BuildImage`, `BuildQCOW2`, `GetJob` and `ListJobs` are ordinary gRPC routes —
Supernova speaks gRPC and registers reflection, so no passthrough handler is
needed. `BuildImage` carries the build context as base64 in the JSON body
(the proto `bytes` field); the gateway's gRPC client and Supernova's server
both raise the message limit to 64 MiB (`client/client_registry.go`,
`supernova/cmd/supernova/main.go`). The routes require a token and are not in
`no_auth_config.yml`; Supernova enforces `image.build|image|<name>`.

## Bugs, disabled and unfinished features

### Confirmed bugs

| Where | Effect |
|---|---|
| `client/client_interceptor.go`, `client/client_rate_limiter.go` | A **new gRPC connection to Apollo and to the rate limiter is dialed on every request** (no connection reuse, no keepalive). Works, but adds latency and file-descriptor churn under load. |
| `client/client_interceptor.go` | Apollo's address is hardcoded `apollo:8002` (and the rate limiter's `rate_limiter_service:8080`); the gateway cannot run outside the compose network without code changes. |
| `Dockerfile` comment | The `sysrl` system-wide limiter exists but is commented out/disabled in the image (deliberate, but undocumented in code paths). |

### Disabled / unfinished features

- **No tests at all** (`go test ./...` → no test files) for routing, auth
  forwarding, or rate-limit behavior.
- Some route definitions are commented out (`DeleteRateLimiter`,
  `CanRateLimiterAllowRequest`) — those operations are reachable only
  internally.
- GET-with-body clients (`cockpit image status`, `vms list/status`) work only
  because this gateway forwards GET bodies; stricter proxies in front of it
  would break them (review BUG-3).

### PoC boundaries (by design, not bugs)

- Plain HTTP + plaintext gRPC internally; no TLS anywhere in the lab.
- Authorization is enforced by downstream services; hitting a service gRPC
  port directly bypasses the gateway (lab networks only).
