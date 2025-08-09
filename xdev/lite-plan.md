## Goal
Make backend runnable as a single binary by making external services optional. Add embedded, drop-in compatible implementations without changing existing `infra/contract` interfaces or breaking tests:
- **DB**: SQLite
- **KV**: Badger (dgraph-io/badger/v4)
- **Search (text + vector)**: DuckDB
- **Message Queue**: Go CDK mempubsub (in-memory)
- **Object Storage**: Go CDK fileblob (local filesystem)

## Non-goals / Constraints
- Do not modify IDs, DTOs, APIs, or `backend/infra/contract/*` interfaces.
- Keep existing tests green; add new tests for embedded providers only.
- Default behavior remains current (e.g., MySQL/ES/etc.). "Lite" mode opt-in via config/flag.
- Keep single-binary friendly dependencies (avoid external daemons). Prefer pure-Go where practical.

## High-level Approach
1. Introduce embedded provider implementations under `backend/infra/impl/*` so they satisfy existing `infra/contract/*` interfaces.
2. Add provider selection via configuration (URIs) + `--lite` flag + `COZE_LITE=1` env.
3. Centralize provider wiring in bootstrap to choose impls by URI scheme (no app-layer changes).
4. Provide local data dir layout `./var/data/{sqlite,nutsdb,duckdb,blob}/` for persistence.
5. Add Make targets for build/run in lite mode; keep Docker optional.

### Priority: Message Queue (Phase 1)
- Implement `infra/impl/mq/gocdk` using `gocloud.dev/pubsub/mempubsub`.
- Config: `mq.uri` accepts either `nsq://...` (current) or `mem://` (lite).
- Documentation only (no file edits here):
  - In `docker/docker-compose.yml`, NSQ services are optional when `mq.uri=mem://`.
  - In `docker/.env.example`, add `MQ_URI=nsq://nsqd:4150?lookupd=http://nsqlookupd:4161` and comment to switch to `MQ_URI=mem://` for lite.
  - `coze-server` should read `MQ_URI` and optional `COZE_LITE` envs.

## Mapping: Contract → Embedded Provider
- **SQL DB** (`infra/contract/db`): `infra/impl/db/sqlite` using `modernc.org/sqlite` for CGO-free build. DSN: `file:./var/data/sqlite/app.db`.
- **KV** (`infra/contract/kv`): `infra/impl/kv/badger` using `github.com/dgraph-io/badger/v4` with one DB per service, logical buckets by key prefix.
- **Search (text + vector)** (`infra/contract/search`): `infra/impl/search/duckdb` using DuckDB. Store documents + embeddings; provide BM25/text search and cosine similarity. File: `./var/data/duckdb/search.db`.
- **MQ** (`infra/contract/mq`): `infra/impl/mq/gocdk` using `gocloud.dev/pubsub/mempubsub` with URIs `mem://topic`.
- **Blob Storage** (`infra/contract/blob`): `infra/impl/blob/gocdk` using `gocloud.dev/blob/fileblob` with URIs like `file:///absolute/path` (default `./var/data/blob`).

## Config & Wiring
- Add new config keys (non-breaking; defaults preserve current behavior):
  - `db.uri`: e.g., `mysql://...` (default) or `sqlite://./var/data/sqlite/app.db`
  - `kv.uri`: e.g., `redis://...` (default) or `badger://./var/data/badger`
  - `search.uri`: e.g., `es://...` (default) or `duckdb://./var/data/duckdb/search.db`
  - `mq.uri`: e.g., `nats://...` (default) or `mem://`
  - `blob.uri`: e.g., `s3://...` (default) or `file://./var/data/blob`
- Add `lite: true|false` as preset switch to override URIs to embedded variants.
- Add `--lite` CLI flag and `COZE_LITE=1` env to set the preset at startup.
- Bootstrap logic reads URIs and instantiates the right impl based on scheme.

## Docker compose/env reference (read-only; no edits here)
- Current compose services: `mysql`, `redis`, `elasticsearch`, `minio`, `etcd`, `milvus`, `nsqlookupd`, `nsqd`, `nsqadmin`, `coze-server`, `coze-web`.
- Lite replacements (toggle via URIs/env; external services become optional):
  - **MySQL → SQLite**
    - External: `mysql://mysql:3306` (compose service `mysql`)
    - Lite: `sqlite://./var/data/sqlite/app.db`
  - **Redis → Badger**
    - External: `redis://redis:6379`
    - Lite: `badger://./var/data/badger`
  - **Elasticsearch → DuckDB (text+vector)**
    - External: `es://elasticsearch:9200`
    - Lite: `duckdb://./var/data/duckdb/search.db`
  - **Milvus (depends on etcd + minio) → DuckDB (vector) + fileblob**
    - External: `milvus://milvus:19530` (+ `etcd`, `minio`)
    - Lite: `duckdb://./var/data/duckdb/search.db` (vector search stored locally) and `file://./var/data/blob`
  - **NSQ (nsqlookupd/nsqd/nsqadmin) → Go CDK mempubsub**
    - External: `nsq://nsqd:4150?lookupd=http://nsqlookupd:4161`
    - Lite: `mem://`
  - **MinIO/S3 → Go CDK fileblob**
    - External: `s3://...` (compose service `minio` at `http://minio:9000`)
    - Lite: `file://./var/data/blob`

- Suggested env keys (to document/create in `docker/.env.example`):
  - `DB_URI=mysql://mysql:3306` or `sqlite://./var/data/sqlite/app.db`
  - `KV_URI=redis://redis:6379` or `badger://./var/data/badger`
  - `SEARCH_URI=es://elasticsearch:9200` or `duckdb://./var/data/duckdb/search.db`
  - `MQ_URI=nsq://nsqd:4150?lookupd=http://nsqlookupd:4161` or `mem://`
  - `BLOB_URI=s3://minio/<bucket>` or `file://./var/data/blob`
  - `COZE_LITE=0|1` (when `1`, preset above URIs to lite defaults)

## Data & Migrations
- SQLite: reuse existing migration engine; add SQLite migrations if dialect differences exist. Ensure WAL mode and pragmatic pragmas for durability.
- DuckDB: create schema for text/vector tables. Provide a backfill command to index from source-of-truth (DB/ES) when present; otherwise index incrementally on writes.
- Badger: logical bucket naming via key prefixes; TTL via Badger entry TTL; lists and hashes via composite keys.
- Blob: create base directory on startup; validate permissions.

## SQL Parser and Dialects (SQLite, PostgreSQL)
- Keep `backend/infra/contract/sqlparser` interface unchanged. MySQL/TiDB-based impl remains for MySQL only.
- Use native parsers for SQLite and PostgreSQL in Go, provided they can implement `SQLParser` and pass all existing tests in `backend/infra/impl/sqlparser/sql_parser_test.go` without modifications.

- Candidates and approach:
  - SQLite (pure Go preferred):
    - Generate a Go parser via ANTLR4 using the official SQLite grammar, vendored under `backend/infra/impl/sqlparser/sqlite/internal/grammar`.
    - Implement `backend/infra/impl/sqlparser/sqlite` that constructs a minimal AST sufficient for required operations (table/column rename, table name extraction, add insert columns, append WHERE) and restores SQL matching test expectations.
  - PostgreSQL:
    - Option A: `github.com/pganalyze/pg_query_go` for parse/AST/pretty-print (CGO likely). Acceptable if we can still produce single binaries for Linux targets.
    - Option B: ANTLR4 PostgreSQL grammar → pure-Go parser with the same minimal AST transforms and SQL restoration.

- Output normalization:
  - Implement a formatter layer to align output strings with current tests (spacing, quoting, lowercasing in expected places observed in tests), so implementations pass without altering tests.

- Selection:
  - Choose implementation by `db.uri` scheme: `sqlite://` → SQLite parser, `postgres://` → PostgreSQL parser, `mysql://` → existing TiDB parser.

- Acceptance:
  - Must pass all existing tests in `sql_parser_test.go` unchanged per dialect implementation.
  - If the first-choice native library cannot satisfy tests, switch to the alternative (ANTLR vs pg_query_go) and iterate until tests pass; do not fall back to MySQL-compatible rewriting for SQLite/PostgreSQL.

## Testing Strategy (keep existing tests intact)
- Add provider conformance tests per contract, shared via interface-driven test suites:
  - DB conformance (subset relevant to app usage)
  - KV conformance (get/set/range/TTL)
  - Search conformance (index, query text, query vector, pagination)
  - MQ conformance (publish/subscribe, ordering/at-least-once guarantees as documented)
  - Blob conformance (put/get/list/delete, signed URLs if applicable → skipped or stubbed)
- Add E2E smoke test in lite preset to boot the full app and hit critical endpoints.

## Build & Run
- Makefile additions:
  - `make build-lite`: build single binary with lite default preset baked (or flag at runtime)
  - `make run-lite`: create `./var/data/*` and run with `--lite`
  - `make test-lite`: run conformance + E2E in lite mode
- Docker remains optional; do not require external services for lite workflows.

## Observability & Ops
- Health checks: expose health endpoints for each provider impl.
- Metrics/logging: reuse existing hooks; label metrics with provider type (e.g., `db_provider=sqlite`).
- Backup/restore: document copying `./var/data/*` directories; provide helper scripts.

## Risks & Mitigations
- DuckDB likely requires CGO; accept CGO for Linux builds or gate behind build tag `duckdb`. Provide fallback `sqlite`-based simple search if unavailable.
- SQL dialect differences (MySQL → SQLite): add migration dialect shims and query adapters where queries are infra-owned.
- Concurrency/locking: configure SQLite WAL, Badger options; document safe limits.
- Search parity vs Elasticsearch: scope to features used by app; document any behavior differences.

## Milestones
1) Scaffolding
   - Add URI-based config + bootstrap wiring
   - Data dir creation utilities
2) Providers
   - MQ: Go CDK mempubsub
   - SQLite impl + migrations
   - Badger KV impl
   - DuckDB impl (text + vector)
   - Blob (file)
3) Tests
   - Conformance suites per contract
   - E2E lite smoke test
4) Tooling & Docs
   - Make targets, runbook, troubleshooting
   - Docker compose: mark external deps optional

## Acceptance Criteria
- App starts with `--lite` and no external services running.
- Core flows pass E2E smoke tests in lite mode.
- All existing tests remain green with default providers.
- Switching providers via URIs requires no code changes outside configuration.

## Action Items (Incremental)
- [ ] Add config URIs + `--lite`/`COZE_LITE` preset handling
- [ ] MQ: implement `infra/impl/mq/gocdk` (mempubsub) and wire
- [ ] Document `docker-compose` and `docker/.env.example` changes (no edits here)
- [ ] Implement `infra/impl/db/sqlite` and wire
- [ ] Implement `infra/impl/kv/nutsdb` and wire
- [ ] Implement `infra/impl/search/duckdb` and wire
- [ ] Implement `infra/impl/blob/gocdk` (fileblob) and wire
- [ ] Dialect adapters for `sqlparser`: sqlite and postgres selection by `db.uri`
- [ ] Add conformance test suites per contract (incl. sqlparser dialects)
- [ ] Add `make build-lite`, `make run-lite`, `make test-lite`
- [ ] Write runbook and troubleshooting for lite mode


