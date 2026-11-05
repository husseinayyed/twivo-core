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

The cache and database clients provide the building blocks for this flow. A unified lookup helper and its configurable keys, TTLs, capacity, write policy, invalidation, batch behavior, and source behavior are planned; until then, applications can use the lower-level MongoDB and Redis APIs directly.

MongoDB remains the source of truth unless an application defines another source. Circuit breakers protect external operations and allow optional cache layers to degrade gracefully.

## Twivo Core roadmap

Twivo Core is intended to grow into a shared data-access and infrastructure toolkit. Planned capabilities include:

- Simple `Set` and `Get` operations across the LRU, Redis, and MongoDB layers.
- Batch reads and writes for efficient multi-item operations.
- Query builders for composing common database queries without repeating boilerplate.
- Cache invalidation, TTL, and write-through/write-back policies.
- A persistent Cuckoo filter to track keys that may exist in the cache or database.
- Additional reusable helpers for common application infrastructure needs.

These are roadmap items; availability and API details may change as each feature is implemented. The lower-level database clients remain available for use in the meantime.

### Planned application data workflow

The goal is to let an application use one API for common data operations while Twivo Core coordinates the appropriate storage layers:

1. **Initialize:** load application configuration, connect to required services, and prepare cache/index state. Startup must treat an absent or untrusted Cuckoo snapshot as incomplete; recovery and rebuild details are still open design work.
2. **Read one item:** use a trusted negative filter result to return not found early. Otherwise check LRU, Redis, and then the configured source of truth. Promote successful lower-layer reads into the faster cache layers and update the key index.
3. **Write one item:** validate and write to the configured source of truth, then update or invalidate cache entries and the key index. The exact write-through/write-back policy and behavior when a cache/index update fails are not yet fixed.
4. **Read or write many items:** batch operations should reduce round trips, preserve per-item outcomes, and follow the same consistency rules as single-item operations.
5. **Build queries:** a future query-builder API should compose filters, projections, sorting, and limits while retaining an escape hatch to native MongoDB/Redis APIs.
6. **Shut down:** close external clients cleanly and attempt to persist the filter snapshot. Periodic snapshotting and shutdown persistence are proposals, not implemented guarantees.

These steps describe the intended direction, not a currently available end-to-end API. Applications should use the existing lower-level clients until each higher-level operation is implemented and documented.

### Cuckoo filter and key-existence hints

The planned Cuckoo filter is a compact index of keys, not a store for values or a replacement for MongoDB. Its purpose is to avoid unnecessary cache and database reads when a key is known to be absent. The intended read path is:

1. Ask the filter about the fully-qualified key.
2. If the filter says the key is absent and the filter is known to be complete and current, return a not-found result without checking the caches or database.
3. If the filter says the key may exist, check the LRU cache, then Redis, then MongoDB according to the configured data flow.
4. When a record is found in the database, add its key to the filter and populate the configured cache layers.

A positive filter result means “possibly present,” not “the record exists”; false positives are expected and must fall through to normal lookup. A negative result can only be used to skip reads when all relevant keys have been indexed and updates are being tracked reliably. The filter cannot establish that a key has never existed historically: deletion, missed writes, stale state, or an incomplete rebuild can make a key absent from the filter even if it existed before. MongoDB (or the configured source of truth) remains authoritative.

Keys should be namespaced to avoid collisions between tenants, collections, or entity types—for example, including the collection/type and tenant identifier as part of the filter key. Insert, update, and delete paths must keep the filter consistent with the source of truth. The exact policy for deleting keys, concurrent writes, and coordinating filter updates with database transactions is still to be designed.

#### Persistence and recovery (proposed)

- Persist a timestamped filter snapshot about once per minute, and attempt a final save during graceful shutdown.
- Store enough metadata to validate a snapshot on startup, such as a format version, save time, and filter configuration. The storage location and exact metadata format are undecided.
- Treat missing, corrupt, incompatible, or too-old snapshots as unavailable; do not use them to make definitive negative decisions.
- Consider rebuilding by scanning the source database and deriving the indexed keys. The scan strategy, startup behavior, and whether lookups can safely proceed while rebuilding have not been decided.
- If persistence or filter updates fail, prefer a slower cache/database lookup over incorrectly returning “not found.”

The one-minute interval and shutdown save are proposed behavior, not a current guarantee. Snapshot durability, write frequency, expiration/deletion handling, and recovery guarantees remain future design decisions.

## Components

| Component | Responsibility | Status |
| --- | --- | --- |
| `pkg/breaker` | Shared process-wide circuit breakers | Available |
| `pkg/db/mongo` | MongoDB client lifecycle and operations | Available |
| `pkg/db/redis` | Redis client lifecycle and operations | Available |
| `pkg/twivo` | Core coordination and shared data-access APIs | Under development |
| `pkg/cache` | Shared LRU/Redis cache workflows and Cuckoo key index | Planned |
| Query builder | Compose common database query options | Planned; package/API TBD |
| `pkg/bootstrap` | Composition layer for initializing infrastructure | Available |
| `infra` | Docker, Compose, and environment configuration | Available |

Packages stay focused on their own concern. Bootstrap code connects them for applications that prefer a single initialization point.

## Requirements

- Go 1.27.3 or newer
- Docker Engine and Docker Compose
- Git
- MongoDB and Redis for integration workflows

## Configuration

```bash
cp infra/.env.example infra/.env
```

Configure `MONGO_USER`, `MONGO_PASSWORD`, `MONGO_URL`, `REDIS_PASS`, and `REDIS_URL`. `REDIS_URL` is the address only (for example, `redis://redis:6379/0`); bootstrap applies `REDIS_PASS` using Redis password authentication. Docker Compose uses the `mongodb` and `redis` service hostnames; local overrides may use `localhost`.

These are the current connection settings. Cache capacities, per-operation TTLs, batch limits, query-builder options, and Cuckoo snapshot location/frequency will need explicit configuration as those APIs are designed; they are not established by this environment setup yet.

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

`make dev` starts the development services; it does not yet imply that the planned unified Set/Get, batch, query-builder, or persistent-filter workflows are active. Use the existing package-level client APIs for operations that are not implemented by Twivo Core yet.

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

When persistent key-index support is implemented, its lifecycle will also need to restore and validate snapshots, distinguish complete from rebuilding state, periodically persist changes, and flush on graceful shutdown. If the index is incomplete or unavailable, reads must fall back to the cache/database path rather than treating an absent key as definitive.

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

As the planned data layer is added, tests should cover cache hits and misses, fallback ordering, batch partial failures, query construction, filter false positives, stale/corrupt snapshots, rebuild behavior, and consistency across successful writes and deletes.

## Status

Twivo Core is under active development. MongoDB, Redis, bootstrap, and breaker helpers are available; unified cache/data operations, batch APIs, query builders, and durable Cuckoo-filter workflows are roadmap items. Public APIs, persistence guarantees, and configuration may evolve as those designs become stable.

## License

This project is licensed under the MIT License. See [LICENSE](LICENSE) for the full text.