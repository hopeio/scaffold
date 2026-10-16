# datasource

One subdirectory per data source, one Go package each: `duckdb`, `flightsql`,
`influxdb`, `meilisearch`.

Each package exposes the same two types, and the package name is the namespace,
so the types stay short:

```go
Duck  duckdb.DB             `init:"config:duckdb"`
Meili meilisearch.Client    `init:"config:meilisearch"`
```

- `<pkg>.Config` — connection settings plus `BeforeInject`/`AfterInject`/`Build()`
- `<pkg>.Client` or `<pkg>.DB` — the injectable host with `Config()`/`Init()`/`Close()`

Why not one flat `package datasource`: it erases the per-source package name, so
every type has to carry its product as a prefix (`DuckDBConfig`, `FlightSQLDB`),
and importing one source links all of them -- DuckDB needs CGO plus per-platform
static libraries, FlightSQL pulls Apache Arrow. Splitting by package keeps both
problems at bay: a binary that only wants MeiliSearch never pays for Arrow.
