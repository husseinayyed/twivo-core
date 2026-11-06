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

### Current cache flow

```text
Application
    │
    ▼
Twivo cache API ──miss──▶ Redis
    │                       │
    └──── populate local LRU on Redis hit
```

The `pkg/twivo` package exposes `SetCache` and `GetCache`. It checks the process-local sharded LRU first, then Redis on a miss, and stores Redis results back in the local cache. Writes update the local shard and attempt to write to Redis. MongoDB is not part of this cache lookup path; use its client APIs directly when database access or cache-miss population from MongoDB is required. The Cuckoo filters and LRU shards are in-memory and are not shared across application instances.

MongoDB remains the source of truth unless an application defines another source. Circuit breakers protect external operations and allow optional cache layers to degrade gracefully.

## Twivo Core roadmap

Twivo Core is intended to grow into a shared data-access and infrastructure toolkit. Planned capabilities include:

- Batch reads and writes for efficient multi-item operations.
- Query builders for composing common database queries without repeating boilerplate.
- Configurable cache invalidation and write-through/write-back policies.
- Persistent Cuckoo-filter snapshots and recovery.
- Additional reusable helpers for common application infrastructure needs.

These are roadmap items; availability and API details may change as each feature is implemented. The lower-level database clients remain available for use in the meantime.

### Cache API behavior and scope

The current cache API provides single-key operations:

1. **Write:** `twivo.SetCache(ctx, key, value, ttl)` updates the local shard and attempts a Redis write. The TTL is in seconds. Both cache operations use a three-second timeout while respecting cancellation or an earlier deadline from `ctx`; a nil context is treated as `context.Background()`.
2. **Read:** `twivo.GetCache(ctx, key)` checks the local shard, then Redis; a Redis hit is added to the local shard. It returns the context error if the operation is canceled or times out.
3. **Use MongoDB separately:** cache misses do not currently fall through to MongoDB, and there is no automatic database-to-cache population workflow.

The TTL applies to the Redis entry; local LRU entries are managed in process memory. The shard manager also exposes lower-level `Read`, `Write`, `ReadLRU`, `WriteLRU`, `ReadCuckoo`, and `WriteCuckoo` operations from `pkg/shard`. Batch APIs, query builders, persistence, and coordinated cache invalidation remain future work.

### In-memory Cuckoo filters and key-existence hints

The shard manager maintains in-memory Cuckoo filters alongside its LRU shards. A filter is a compact key index, not a store for values or a replacement for MongoDB. The current cache read path uses it to avoid an LRU lookup when a key is not indexed:

1. Ask the filter about the fully-qualified key.
2. If the filter says the key is absent, the local shard lookup returns not found and the `pkg/twivo` API proceeds to Redis.
3. If the filter says the key may exist, check the local LRU; on a miss, the `pkg/twivo` API proceeds to Redis.
4. A successful cache write records the key in the local filter and stores the value in the local LRU.

A positive filter result means “possibly present,” not “the value exists”; false positives are expected and must fall through to the LRU lookup. The filters are process-local, are not persisted, and are not synchronized between replicas. Do not treat them as authoritative for database existence or use them as a replacement for the configured source of truth. MongoDB (or another application datastore) remains the authority for durable records.

Keys should be namespaced to avoid collisions between tenants, collections, or entity types—for example, including the collection/type and tenant identifier as part of the cache key. The current API does not provide coordinated deletion or database transaction handling.

#### Persistence and recovery (not implemented)

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
| `pkg/twivo` | Single-key local LRU and Redis cache operations | Available; under development |
| `pkg/shard` | Process-local sharded LRU caches and Cuckoo filters | Available; under development |
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

`make dev` starts the development services. The current `pkg/twivo` cache API uses the local shard and Redis; batch, query-builder, MongoDB fallback, and persistent-filter workflows are not implemented. Use the package-level database clients for operations outside the cache API.

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

The current Cuckoo filters are in-memory only. Persistent key-index support will need to restore and validate snapshots, distinguish complete from rebuilding state, periodically persist changes, and flush on graceful shutdown. If the index is incomplete or unavailable, reads must fall back to the cache/database path rather than treating an absent key as definitive.

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

Twivo Core is under active development. MongoDB, Redis, bootstrap, and breaker helpers are available. Single-key local LRU/Redis cache operations and process-local Cuckoo-filter sharding are implemented; batch APIs, query builders, MongoDB cache fallback, and durable Cuckoo-filter workflows remain roadmap items. Public APIs, persistence guarantees, and configuration may evolve as those designs become stable.

## License

This project is licensed under the MIT License. See [LICENSE](LICENSE) for the full text.