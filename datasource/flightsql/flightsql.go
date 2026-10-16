/*
 * Copyright 2026 hopeio. All rights reserved.
 * Licensed under the MIT License that can be found in the LICENSE file.
 * @Created by jyb
 */

package flightsql

import (
	"database/sql"

	_ "github.com/apache/arrow-adbc/go/adbc/sqldriver/flightsql"
)

// Config opens an Arrow Flight SQL endpoint through the ADBC driver.
type Config struct {
	// DSN is the flightsql URI, e.g. grpc://host:port?username=...&password=...
	DSN string
}

// BeforeInject is a no-op satisfying the config injection contract.
func (c *Config) BeforeInject() {}

// AfterInject is a no-op satisfying the config injection contract.
func (c *Config) AfterInject() {}

// Build opens a FlightSQL database via the registered ADBC driver. The returned
// handle is lazy: sql.Open only validates driver presence, it does not connect.
func (c *Config) Build() (*sql.DB, error) {
	return sql.Open("flightsql", c.DSN)
}

// DB is the injectable host around a FlightSQL connection pool.
type DB struct {
	*sql.DB
	Conf Config
}

// Config returns the embedded Conf as the configuration for injection.
func (m *DB) Config() any {
	return &m.Conf
}

// Init opens the FlightSQL database and stores it.
func (m *DB) Init() error {
	var err error
	m.DB, err = m.Conf.Build()
	return err
}

// Close closes the FlightSQL database connection if it was initialized.
func (m *DB) Close() error {
	if m.DB == nil {
		return nil
	}
	return m.DB.Close()
}
