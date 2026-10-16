/*
 * Copyright 2026 hopeio. All rights reserved.
 * Licensed under the MIT License that can be found in the LICENSE file.
 * @Created by jyb
 */

package datasource

import (
	"database/sql"
	"path/filepath"
	"strings"
	"testing"
)

// Both sql drivers must stay registered by the blank imports in this package:
// dropping one during a refactor only shows up at runtime as "unknown driver".
func TestSQLDriversRegistered(t *testing.T) {
	drivers := strings.Join(sql.Drivers(), ",")
	for _, want := range []string{"duckdb", "flightsql"} {
		if !strings.Contains(drivers, want) {
			t.Errorf("driver %q not registered (drivers: %s)", want, drivers)
		}
	}
}

func TestDuckDBBuild(t *testing.T) {
	c := DuckDBConfig{DSN: filepath.Join(t.TempDir(), "duck.db") + "?access_mode=read_write&threads=4"}
	db, err := c.Build()
	if err != nil {
		t.Fatal("Build err", err)
	}
	defer db.Close()
	if _, err := db.Exec("CREATE TABLE people (id INTEGER, name VARCHAR)"); err != nil {
		t.Fatal("Exec err", err)
	}
}

// Host lifecycle: Init builds a handle, Close is safe both after Init and on a
// zero-value host.
func TestHostLifecycle(t *testing.T) {
	duck := &DuckDB{Conf: DuckDBConfig{DSN: ":memory:"}}
	if err := duck.Init(); err != nil {
		t.Fatal("DuckDB.Init", err)
	}
	if duck.DB == nil {
		t.Fatal("DuckDB.Init left no handle")
	}
	if err := duck.Close(); err != nil {
		t.Fatal("DuckDB.Close", err)
	}
	if err := (&DuckDB{}).Close(); err != nil {
		t.Errorf("Close on an uninit host should be a no-op, got %v", err)
	}

	// Neither HTTP client touches the network while building, so an unreachable
	// endpoint must still init cleanly.
	influx := &InfluxDBClient{Conf: InfluxDBConfig{ServerURL: "http://127.0.0.1:1"}}
	if err := influx.Init(); err != nil {
		t.Errorf("InfluxDBClient.Init: %v", err)
	}
	if influx.Client == nil {
		t.Error("InfluxDBClient.Init left no client")
	}
	if err := influx.Close(); err != nil {
		t.Errorf("InfluxDBClient.Close: %v", err)
	}

	meili := &MeiliSearchClient{Conf: MeiliSearchConfig{Host: "http://127.0.0.1:1", APIKey: "k"}}
	if err := meili.Init(); err != nil {
		t.Errorf("MeiliSearchClient.Init: %v", err)
	}
	if meili.Service == nil {
		t.Error("MeiliSearchClient.Init left no service")
	}
	if err := meili.Close(); err != nil {
		t.Errorf("MeiliSearchClient.Close: %v", err)
	}
}

// The hosts must keep satisfying the injection contract of the bootstrap layer
// without importing it. These are compile-time proofs: a drifting Config()/
// Init()/Close() signature stops this package from building instead of failing at
// inject time inside some application repo.
type hostContract interface {
	Config() any
	Init() error
	Close() error
}

type configHooks interface {
	BeforeInject()
	AfterInject()
}

var (
	_ hostContract = (*DuckDB)(nil)
	_ hostContract = (*FlightSQLDB)(nil)
	_ hostContract = (*InfluxDBClient)(nil)
	_ hostContract = (*MeiliSearchClient)(nil)

	_ configHooks = (*DuckDBConfig)(nil)
	_ configHooks = (*FlightSQLConfig)(nil)
	_ configHooks = (*InfluxDBConfig)(nil)
	_ configHooks = (*MeiliSearchConfig)(nil)
)
