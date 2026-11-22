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
default token bucket is auto-created with heliosphere's
`RL_DEFAULT_REQ_PER_SEC` / `RL_DEFAULT_BURST` (1/1 when unset; the compose stack sets
10/20 in `tools/.env`). Over the limit the reply is `User reached rate limit
per request.` Limiters already stored in Consul keep their old values; delete
them (`consul kv delete -recurse rateLimiters/`) after raising the default,
or create a custom limiter for the user/method (`/rate-limiter` no-auth
routes).

A second, system-wide limiter can be enabled by starting the binary with the
`sysrl` argument (disabled in the compose deployment; see `Dockerfile`).

## Configuration

| File | Purpose |
|---|---|
| `config/config.yml` | services (host:port) + route table (52 routes today) |
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
go build ./...       # no tests exist
go run .             # needs config.yml + no_auth_config.yml in CWD
                     # and the target services reachable at compose hostnames
```

The Dockerfile builds from the **workspace root** context (it copies
`oort/`, `magnetar/`, `apollo/`, `heliosphere/` next to the gateway) and runs
`./main` on port 5555. In compose it is the `lunar-gateway` container,
exposed on `localhost:5555`.
