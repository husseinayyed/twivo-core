# Twivo Core

**Shared infrastructure and reusable helpers for the Twivo ecosystem.**

Twivo Core centralizes common Go infrastructure used across Twivo applications: MongoDB, Redis, circuit breakers, cache orchestration, configuration, and containerized environments. It combines complete, ready-to-use workflows with low-level building blocks, so applications save integration time without losing the ability to configure or control behavior manually.

## Why Twivo Core?

- Build complete workflows once and reuse them across every Twivo application.
- Update shared infrastructure from one package instead of many repositories.
- Apply consistent connection, timeout, retry, cache, logging, and circuit-breaker behavior.
- Support fast, process-local LRU access → shared Redis caching → durable MongoDB reads.
- Start new applications quickly while still bypassing helpers for custom strategies.
- Keep database and resilience logic separate from application business logic.

### Planned data flow

```text
Application
    │
    ▼
LRU memory  ──miss──▶  Redis  ──miss──▶  MongoDB
    ▲                    ▲                    │
    └──────── fill from Redis/MongoDB ────────┘
```

A lookup helper can perform this complete flow and populate the cache layers. Its keys, TTLs, capacity, write policy, invalidation, batch behavior, and source behavior remain configurable. Lower-level MongoDB and Redis APIs remain available for custom workflows.

MongoDB remains the source of truth unless an application defines another source. Circuit breakers protect external operations and allow optional cache layers to degrade gracefully.

## Components

| Component | Responsibility |
| --- | --- |
| `pkg/breaker` | Shared process-wide circuit breakers |
| `pkg/db/mongo` | MongoDB client lifecycle and operations |
| `pkg/db/redis` | Redis client lifecycle and operations |
| `pkg/cache` | Shared LRU and Redis cache abstractions |
| `pkg/bootstrap` | Composition layer for initializing infrastructure |
| `infra` | Docker, Compose, and environment configuration |

Packages stay focused on their own concern. Bootstrap code connects them for applications that prefer a single initialization point.

## Requirements

- Go 1.26 or newer
- Docker Engine and Docker Compose
- Git
- MongoDB and Redis for integration workflows

## Configuration

```bash
cp infra/.env.example infra/.env
```

Configure `MONGODB_USER`, `MONGODB_PASSWORD`, `MONGODB_URL`, `REDIS_URL`, and `REDIS_PASS`. Docker Compose uses the `mongodb` and `redis` service hostnames; local overrides may use `localhost`.

Never commit real credentials. Inject production secrets through the deployment platform or a secret manager.

## Development workflow

A Makefile will provide the primary workflow:

```text
make dev       Start development infrastructure
make test      Run tests
make lint      Run formatting and static checks
make build     Build the module
make logs      Show service logs
make down      Stop infrastructure containers
make clean     Remove generated artifacts
```

Until these targets exist, run the Compose development environment directly:

```bash
docker compose \
  -f infra/docker-compose.yaml \
  -f infra/docker-compose.dev.yaml \
  --env-file infra/.env up --build

docker compose \
  -f infra/docker-compose.yaml \
  -f infra/docker-compose.dev.yaml \
  --env-file infra/.env down
```

## Go module usage

```text
github.com/husseinayyed/twivo-core
```

Applications can consume individual packages or the shared bootstrap API. Prefer explicit dependency injection for testable code and typed APIs for public clients and operations:

```go
services, err := bootstrap.Init()
if err != nil {
    return err
}
defer services.Close(context.Background())
```

High-level helpers provide convenience and standard policy. Lower-level APIs remain available for custom cache, database, retry, fallback, and invalidation behavior.

## Circuit breakers

```go
breaker.Init()

result, err := breaker.Group.Execute("Mongo", func() (any, error) {
    return connectMongo()
})
```

Named breakers protect external dependencies. State changes should be visible through structured logs and metrics, with configurable thresholds and timeouts.

## Lifecycle

Infrastructure clients should:

1. Validate configuration.
2. Connect using bounded contexts.
3. Verify connectivity.
4. Reuse connection pools.
5. Execute external calls behind circuit breakers.
6. Close cleanly during application shutdown.

Reusable packages should return errors instead of terminating the process. Avoid mutable package-level clients and long-lived operations without context cancellation.

## Development guidelines

- Keep each package focused on one infrastructure concern.
- Return errors and let applications decide how to handle them.
- Make helper workflows configurable and provide lower-level escape hatches.
- Use interfaces when provider-independent behavior improves testing.
- Keep credentials outside source control.
- Add unit tests for helpers and cache policies.
- Add integration tests for MongoDB and Redis.
- Format Go code and run tests before committing.

## Testing

```bash
go test ./...
```

Integration tests should isolate external services and skip cleanly when dedicated infrastructure is unavailable.

## Status

Twivo Core is under active development. Public APIs and configuration may evolve as shared infrastructure patterns become stable.

## License

License details will be added when a project license is selected.