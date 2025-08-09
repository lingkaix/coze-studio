## Lite mode (single-binary) guide

Lite mode makes external services optional by swapping providers for embedded, local implementations. It is enabled via a CLI flag or environment variables and works without changing any application-layer code.

### Enable lite mode

- CLI: start the server with the flag:

  ```sh
  ./coze-server --lite
  ```

- Environment variable: set either of these before startup:
  - `LITE_MODE=1`
  - `COZE_LITE=1`

- Makefile helpers:
  - `make build-lite` – build the server (debug flags applied)
  - `make run-lite` – creates local data dirs and runs with lite defaults
  - `make test-lite` – runs conformance/smoke tests for lite providers
  - `make clean-lite-test-data` – removes `./var/data/{duckdb,badger}`

Lite presets do not override explicitly provided env vars; they only fill in defaults.

### Environment variables in lite mode

- DB selection: `DB_URI`
  - Examples:
    - `sqlite://./var/data/sqlite/app.db` (lite default)
    - `mysql://user:pass@host:3306/dbname?parseTime=true` (translated to `MYSQL_DSN` internally)
  - Effect: chooses the SQL provider
    - `sqlite://` → embedded SQLite provider
    - `mysql://` or empty → MySQL provider using existing config

- KV/Cache selection: `KV_URI` or `CACHE_URI`
  - Examples:
    - `badger://./var/data/badger` (lite default if neither KV_URI nor CACHE_URI set)
    - `mem://` (embedded in-memory Redis via miniredis; ephemeral)
  - Effect: chooses the cache provider used by `cache.Cmdable`
    - `badger://<dir>` → Badger-backed local key-value store (persistent under `<dir>`)
    - `mem://` → in-memory Redis stub (good for tests/dev)
    - otherwise → external Redis via `REDIS_ADDR`, `REDIS_PASSWORD`

- Search selection: `SEARCH_URI`
  - Example: `duckdb://./var/data/duckdb/lite.duckdb` (lite default)
  - Effect: selects embedded DuckDB-like search provider (file-backed JSON store). When unset, defaults to external search per existing configuration.

- Message queue selection: `COZE_MQ_TYPE`
  - Example: `mem` (lite default)
  - Effect: uses in-memory pub/sub provider in lite mode. Any explicitly set value is preserved.

- Blob storage selection: `STORAGE_TYPE`, `BLOB_DIR`, `BLOB_URI`
  - Examples:
    - `STORAGE_TYPE=file` and `BLOB_DIR=./var/data/blob` (lite defaults)
    - `BLOB_URI=file:///absolute/path` (explicit file path; takes precedence)
  - Effect: chooses object storage
    - `BLOB_URI=file://...` → local file storage (`fileblob`) at the given path
    - `STORAGE_TYPE=file` → local file storage at `BLOB_DIR`
    - other types: `minio`, `s3`, `tos` use existing external providers
  - Note: Image service (`imagex`) also switches to a file-backed implementation when `STORAGE_TYPE=file` or `BLOB_URI` is set.

### What the lite preset sets by default

When `LITE_MODE=1` or `COZE_LITE=1`, and only if corresponding envs are not already set:

- `DB_URI=sqlite://./var/data/sqlite/app.db`
- `COZE_MQ_TYPE=mem`
- `STORAGE_TYPE=file`
- `BLOB_DIR=./var/data/blob`
- `KV_URI=badger://./var/data/badger` (skipped if `KV_URI` or `CACHE_URI` already set)
- `SEARCH_URI=duckdb://./var/data/duckdb/lite.duckdb`

Data directories are created on demand under `./var/data/{sqlite,badger,duckdb,blob}`.

### Notes

- You can override any default by explicitly setting the corresponding env var.
- For cache in lite mode, prefer `badger://...` for persistence; use `mem://` for ephemeral dev/testing.
- The `--lite` flag is a convenience for setting `LITE_MODE=1` before config/env resolution.


