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

Configure `MONGO_USER`, `MONGO_PASSWORD`, `MONGO_URL`, `REDIS_PASS`, and `REDIS_URL`. `REDIS_URL` is the address only (for example, `redis://redis:6379/0`); bootstrap applies `REDIS_PASS` using Redis password authentication. Docker Compose uses the `mongodb` and `redis` service hostnames; local overrides may use `localhost`.

MongoDB's Compose root user is authenticated against the `admin` database. The MongoDB image only creates that user when its data directory is first initialized; if credentials changed after the dev volume was created, the persisted database still has the old credentials. Update that database user or recreate the Mongo dev volume (this deletes its local data).

Never commit real credentials. Inject production secrets through the deployment platform or a secret manager.

## Development workflow

A Makefile provides the primary workflow:

```text
make dev        Start development app, MongoDB, and Redis
make test       Run tests in Docker with temporary databases
make shell      Enter the development app container
make build-dev  Build the development image
make build-test Build the test image
make build-prod Build the production image
make prod       Start the production app and databases
make logs       Follow development logs
make down       Stop development containers
```

Set up the local environment file before running Compose targets:

```bash
cp infra/.env.example infra/.env
```

## Add infrastructure in two steps

No need to wire MongoDB, Redis, and circuit breakers by hand. Import bootstrap, then initialize the shared clients:

```go
import "github.com/husseinayyed/twivo-core/pkg/bootstrap"

func main() {
    bootstrap.MustRun()
    // Use bootstrap.Mongo_Client, bootstrap.Redis_Client, and bootstrap.Breaker.
}
```

`MustRun()` reads `MONGO_URL`, `MONGO_USER`, `MONGO_PASSWORD`, `REDIS_URL`, and `REDIS_PASS`, connects to MongoDB and Redis, and configures the shared circuit breakers. It exits with a logged error if startup fails, which is convenient for a service entry point. Set those variables in your deployment environment before starting the application.

After initialization, access the clients from bootstrap. For example, select a MongoDB database and use the Redis client:

```go
import (
    "context"
    "time"

    "github.com/husseinayyed/twivo-core/pkg/bootstrap"
)

func run() error {
    if err := bootstrap.Run(); err != nil {
        return err
    }
    defer bootstrap.Mongo_Client.Disconnect(context.Background())
    defer bootstrap.Redis_Client.Close()

    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    appDB := bootstrap.Mongo_Client.Database("myapp")
    users := appDB.Collection("users")
    if _, err := users.InsertOne(ctx, map[string]string{"name": "Ada"}); err != nil {
        return err
    }
    return bootstrap.Redis_Client.Ping(ctx).Err()
}
```

`bootstrap.Breaker` is the shared breaker registry. Its `Execute` method runs an operation behind a named breaker:

```go
result, err := bootstrap.Breaker.Execute("Mongo", func() (any, error) {
    return performMongoOperation()
})
```

If the calling application needs to decide how startup failures are handled, use the error-returning `bootstrap.Run()` instead of `MustRun()`:

```go
if err := bootstrap.Run(); err != nil {
    return err
}
```

The initialized clients are long-lived connection pools; reuse them rather than creating a new client per operation. Close them during application shutdown:

```go
defer bootstrap.Mongo_Client.Disconnect(ctx)
defer bootstrap.Redis_Client.Close()
```

Applications can import `pkg/db/mongo`, `pkg/db/redis`, or `pkg/breaker` directly when they need custom configuration or policies instead of bootstrap's shared defaults.

## Circuit breakers

Bootstrap uses the shared named breakers for MongoDB and Redis connections. Applications that need custom behavior can use `pkg/breaker` directly; breaker state changes are logged, and thresholds/timeouts are configured centrally.

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