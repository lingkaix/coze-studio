### M1 review decisions and M2 action items

#### KV (Badger) packaging and API
- **Decision**: Keep Badger in its own package with a proper exported constructor; avoid cross-dir unexported calls.
- **Actions (tests first)**
  - [ ] Add unit tests for `badger.New(path)` returning a `cache.Cmdable` and exercising Set/Get/Expire/Hash/List/Incr using `t.TempDir()`.
  - [ ] Change `backend/infra/impl/cache/badger` to `package badger` and export:
    - `func New(path string) (cache.Cmdable, error)`
  - [ ] Update `backend/infra/impl/cache/redis/redis.go` to import `impl/cache/badger` and, when `KV_URI`/`CACHE_URI` uses `badger://`, parse the path and call `badger.New(path)`.
  - [ ] Move/adjust tests to match the package and import path; remove cross-dir usage of unexported symbols.
  - [ ] Update `backend/Makefile` to run `go test ./infra/impl/cache/badger` (not `./infra/impl/cache/redis -run TestBadger_`).

#### Test data hygiene (Badger and DuckDB)
- **Goal**: No persistent artifacts under repo `./var/data/*` as a test side-effect.
- **Actions (tests first)**
  - [ ] Badger tests: set `KV_URI` or `CACHE_URI` to `badger://` + `t.TempDir()`; rely on `t.TempDir()` cleanup.
  - [ ] DuckDB tests (`infra/impl/es/duckdb`): replace hardcoded paths under `./var/data/duckdb/*.json` with `dir := t.TempDir(); path := filepath.Join(dir, "test.duckdb.json")` and `t.Cleanup` as needed.
  - [ ] If any test must touch repo-local paths, add `t.Cleanup(func(){ _ = os.RemoveAll(pathOrDir) })` to remove after execution.
  - [ ] Optional: add `make clean-lite-test-data` target to remove `./var/data/duckdb` and `./var/data/badger` (dev/CI use only).

#### SQLite driver direction (align with existing code)
- **Decision**: Standardize on `gorm.io/driver/sqlite` for mainline to align with usages (e.g., `backend/internal/mock/infra/contract/orm`). Keep a CGO-free fallback path available via build tags using `github.com/glebarez/sqlite` if needed for single-binary builds.
- **Parser requirement**: DB driver choice does not satisfy `infra/contract/sqlparser`. We still need a dedicated SQLite dialect implementation for the SQL parser (as planned). Keep parser work independent of the driver.
- **JSON support**: Use SQLite JSON1 extension.
  - Build with tags enabling JSON1: `-tags "json1"`.
  - Add a small runtime test ensuring `json_extract` works.
- **Full Text Search (FTS)**: Use FTS5.
  - Build with tags enabling FTS5: `-tags "sqlite_fts5"`.
  - Create FTS virtual tables via SQL and add a minimal test for `MATCH` queries.
- **Vector search**: Not natively supported in SQLite without third-party extensions (e.g., `sqlite-vec`, `sqlite-vss`) and `load_extension`, which complicates single-binary distribution. Keep vector search in the DuckDB search provider for lite mode; do not rely on SQLite for vector.
- **Build/CI changes**
  - [ ] Enable CGO for server builds using `gorm.io/driver/sqlite` (Docker/CI): `CGO_ENABLED=1`.
  - [ ] Build with tags: `go build -tags "json1 sqlite_fts5" ./...`
  - [ ] Keep a separate build tag (optional) for CGO-free lite builds that use `glebarez/sqlite` inside our `infra/impl/sqlite` wrapper.
  - [ ] Ensure `backend/Dockerfile` installs a C toolchain as needed for CGO.
- **Dependencies cleanup**
  - [ ] Keep `gorm.io/driver/sqlite` (present in `go.mod`).
  - [ ] Remove unused SQLite libs if not required by the selected paths (e.g., `modernc.org/sqlite`, indirect `mattn/go-sqlite3` may remain via gorm; keep only what we actually use).
  - [ ] Run `go mod tidy` and verify builds for both CGO-on and CGO-off variants per our tags.
- **Smoke/regression tests**
  - [ ] Add a smoke test that opens SQLite via `DB_URI=sqlite://...` and executes basic DDL/DML.
  - [ ] Add tiny tests for JSON1 function availability and FTS5 query execution.

#### Misc DX and consistency
- [ ] Add a `--lite` CLI flag to trigger `applyLitePreset()` before env resolution.
- [ ] Align data dir naming in tooling (`badger` not `nutsdb`) and ensure `run-lite` creates `./var/data/badger`.
- [ ] Keep metrics/logs labeled with provider type (e.g., `db_provider=sqlite`, `kv_provider=badger`, `search_provider=duckdb_stub` or `duckdb_native`).

Notes
- Tests-first workflow: commit tests, then implementation edits to satisfy them.
- Do not override explicit envs; `applyLitePreset()` only fills defaults.
- Maintain DDD boundaries; these changes should remain within `infra/*` and bootstrap wiring.

