/*
 * Copyright 2026 hopeio. All rights reserved.
 * Licensed under the MIT License that can be found in the LICENSE file.
 * @Created by jyb
 */

package datasource

import (
	"database/sql"

	_ "github.com/duckdb/duckdb-go/v2"
)

// Building against a system libduckdb instead of the bundled static libraries:
//
//	CGO_ENABLED=1 CGO_LDFLAGS="-L/path/to" CGO_CFLAGS="-I/path/to" \
//	  go build -tags=duckdb_use_lib ./...
//
// unix: LD_LIBRARY_PATH=/path/to/libs ./main
// win:  PATH=/path/to/libs:$PATH, or copy the dll next to the binary
// (https://github.com/marcboeker/go-duckdb/issues/115)

// DuckDBConfig opens an embedded (file path) or in-memory (":memory:") DuckDB.
type DuckDBConfig struct {
	DSN string `json:"dsn"`
}

// BeforeInject is a no-op satisfying the config injection contract.
func (c *DuckDBConfig) BeforeInject() {}

// AfterInject is a no-op satisfying the config injection contract.
func (c *DuckDBConfig) AfterInject() {}

// Build opens a DuckDB database using the standard database/sql interface.
func (c *DuckDBConfig) Build() (*sql.DB, error) {
	return sql.Open("duckdb", c.DSN)
}

// DuckDB is the injectable host around a DuckDB handle.
type DuckDB struct {
	*sql.DB
	Conf DuckDBConfig
}

// Config returns the embedded Conf as the configuration for injection.
func (m *DuckDB) Config() any {
	return &m.Conf
}

// Init opens the DuckDB database and stores it.
func (m *DuckDB) Init() error {
	var err error
	m.DB, err = m.Conf.Build()
	return err
}

// Close closes the DuckDB database connection if it was initialized.
func (m *DuckDB) Close() error {
	if m.DB == nil {
		return nil
	}
	return m.DB.Close()
}
