# Twivo Core

Shared infrastructure for Twivo Go applications.

Twivo Core provides reusable infrastructure for MongoDB, Redis, caching, sharding, and circuit breakers. It centralizes common backend components so applications can share consistent behavior without repeatedly implementing and maintaining the same infrastructure.

## Features

- **Bootstrap:** Centralized initialization and access to shared MongoDB and Redis clients.
- **Circuit Breakers:** Shared, named circuit breakers for dependency protection.
- **Two-Layer Caching:** LRU caching backed by Redis.
- **Cuckoo Filters:** Compact, in-memory key-existence hints for efficient cache lookups.
- **Shard System:** Manages multiple Cuckoo filter shards and their capacity.
- **Cross-Instance Synchronization:** Keeps cache data synchronized between application instances.
- **Automated Testing:** Implemented components and behavior covered by automated tests.
- **Modular Architecture:** Use the high-level API or individual packages independently.

## Before and After Twivo Core

**Before Twivo Core**

Every application independently implements and maintains its infrastructure:

- MongoDB and Redis initialization and connection management.
- Environment configuration and error handling.
- Cache lookup, invalidation, and expiration logic.
- Local caching and Redis fallback.
- Key-existence checks and caching optimizations.
- Sharding and Cuckoo filter management.
- Circuit-breaker configuration and management.
- Infrastructure testing and maintenance.

**After Twivo Core**

Infrastructure is centralized into reusable, tested packages:

- **Bootstrap:** Shared MongoDB and Redis clients with centralized initialization.
- **Cuckoo Filters:** Compact, in-memory key-existence hints that help avoid unnecessary cache lookups.
- **Shard System:** Organizes Cuckoo filters into multiple shards, managing capacity and rotating to new shards when necessary.
- **Sharded LRU:** Fast local caching with efficient memory access.
- **Redis Fallback:** Retrieves values on local cache misses and populates the local LRU on Redis hits.
- **Cross-Instance Synchronization:** Keeps cache data synchronized between application instances.
- **Circuit Breakers:** Shared, named breakers for dependency protection.
- **Automated Testing:** Implemented components and behavior are covered by automated tests.
- **Modular Architecture:** Use the high-level API or individual packages when more control is needed.

**The result:** Less duplicated infrastructure, less boilerplate, consistent behavior across applications, and more development time spent building product features.

## Requirements

- Go 1.27.3 or later.
- Docker Engine and Docker Compose.
- Git.

MongoDB and Redis are required for workflows that use those services.

## Quick Start

Clone the repository:

```bash
git clone https://github.com/husseinayyed/twivo-core.git
cd twivo-core
```

Configure the environment:

```bash
cp infra/.env.example infra/.env
```

Start the development environment:

```bash
make dev
```

## Configuration

Environment variables are loaded from `infra/.env`.

| Variable | Description |
|---|---|
| `MONGO_USER` | MongoDB username |
| `MONGO_PASSWORD` | MongoDB password |
| `MONGO_URL` | MongoDB connection URL |
| `REDIS_PASS` | Redis password |
| `REDIS_URL` | Redis address |

When using Docker Compose, use the service hostnames `mongodb` and `redis` where appropriate. For services running directly on your machine, use `localhost`.

`REDIS_URL` specifies the Redis address. Bootstrap applies the Redis password separately.

**MongoDB note:** The Compose configuration initializes its root user against the `admin` authentication database. MongoDB initializes credentials when the data volume is first created. Changing credentials in `.env` does not automatically update an existing volume.

Keep credentials out of version control.

## Bootstrap

Use the bootstrap package to initialize shared infrastructure.

`bootstrap.Run()` returns an error if initialization fails. Alternatively, `bootstrap.MustRun()` provides fail-fast initialization.

Bootstrap exposes shared clients:

- `bootstrap.Mongo_Client`
- `bootstrap.Redis_Client`
- `bootstrap.Breaker`

Reuse initialized clients instead of creating new connections for every operation. Close clients during application shutdown according to their respective package APIs.

## Caching

The `pkg/twivo` package provides the high-level cache API.

- `twivo.SetCache(ctx, key, value, ttl)`
- `twivo.GetCache(ctx, key)`

The TTL is specified in seconds.

### Cache Read Flow

1. Check the Cuckoo filter when applicable.
2. Use the local sharded LRU for cache lookups.
3. On a local cache miss, retrieve the value from Redis.
4. On a Redis hit, populate the local LRU.
5. Return the result.

Cache writes update the local cache and Redis backing layer.

### Cache Behavior

- Redis provides the shared backing layer.
- Cache data is synchronized between application instances.
- The LRU provides fast local access.
- Cuckoo filters provide compact key-existence hints, not stored values.
- False positives are possible, so a positive filter result does not guarantee that a value exists.
- The cache does not fall back to MongoDB.

Cache operations use a three-second timeout by default. Context cancellation and earlier context deadlines are respected. A nil context is treated as `context.Background()`.

## Cuckoo Filters and Shard System

Cuckoo filters provide compact, in-memory key-existence hints.

They are not value stores or authoritative indexes.

The shard system organizes filters into multiple shards and manages their capacity.

Key characteristics:

- Multiple shards can be maintained as capacity grows.
- New shards are created when the current shard reaches its capacity.
- Existing keys are checked to avoid unnecessary duplicate insertions.
- Negative filter results can help skip unnecessary local cache lookups.
- Positive results still require checking the actual cache.
- False positives are possible.

Cuckoo filters are an optimization layer and should not be treated as the source of truth for cached data.

## Circuit Breakers

Bootstrap exposes a shared circuit-breaker registry through `bootstrap.Breaker`.

Use named breakers for dependencies such as MongoDB and Redis. This allows operations targeting the same dependency to share breaker state.

The lower-level `pkg/breaker` package is also available when an application needs its own breaker registry.

## Lower-Level Packages

Twivo Core is modular. Applications can use individual packages without relying entirely on the high-level API.

| Package | Responsibility |
|---|---|
| `pkg/bootstrap` | Shared infrastructure initialization |
| `pkg/db/mongo` | MongoDB integration |
| `pkg/db/redis` | Redis integration |
| `pkg/breaker` | Circuit-breaker implementation |
| `pkg/twivo` | High-level cache API |
| `pkg/shard` | Sharding and Cuckoo filter functionality |

See the package documentation for exported APIs and implementation-specific details.

## Make Commands

| Command | Description |
|---|---|
| `make dev` | Start the development environment |
| `make test` | Run tests |
| `make shell` | Open the development shell |
| `make build-dev` | Build the development image |
| `make build-test` | Build the test image |
| `make build-prod` | Build the production image |
| `make prod` | Start the production environment |
| `make logs` | Follow environment logs |
| `make down` | Stop the environment |
| `make help` | Display available commands |
| `make build` | Alias for `make build-prod` |

## Testing

Run the Go test suite:

```bash
go test ./...
```

Or use:

```bash
make test
```

Format Go source files:

```bash
gofmt -w .
```

## Project Structure

```text
twivo-core/
├── infra/
│   ├── .env.example
│   └── ...
├── pkg/
│   ├── bootstrap/
│   ├── breaker/
│   ├── db/
│   │   ├── mongo/
│   │   └── redis/
│   ├── shard/
│   └── twivo/
├── Makefile
├── go.mod
└── README.md
```

## Project Status

Twivo Core is being developed as shared infrastructure for the Twivo ecosystem.

The README documents the implemented interfaces and supported workflows. Features still under development should not be considered available until implemented and tested.

## License

See [LICENSE](LICENSE).