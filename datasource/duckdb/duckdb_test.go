/*
 * Copyright 2026 hopeio. All rights reserved.
 * Licensed under the MIT License that can be found in the LICENSE file.
 * @Created by jyb
 */

package duckdb

import (
	"database/sql"
	"path/filepath"
	"strings"
	"testing"
)

// The host and its config must keep satisfying the bootstrap injection contract
// without importing it. Compile-time proof: a drifting signature fails this
// package's build instead of failing at inject time in some application repo.
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
	_ hostContract = (*DB)(nil)
	_ configHooks  = (*Config)(nil)
)

func TestDriverRegistered(t *testing.T) {
	if !strings.Contains(strings.Join(sql.Drivers(), ","), "duckdb") {
		t.Error("duckdb driver not registered by the blank import")
	}
}

func TestBuildAndLifecycle(t *testing.T) {
	c := Config{DSN: filepath.Join(t.TempDir(), "d.db") + "?access_mode=read_write&threads=4"}
	db, err := c.Build()
	if err != nil {
		t.Fatal("Build err", err)
	}
	if _, err := db.Exec("CREATE TABLE people (id INTEGER, name VARCHAR)"); err != nil {
		t.Fatal("Exec err", err)
	}
	db.Close()

	m := &DB{Conf: Config{DSN: ":memory:"}}
	if err := m.Init(); err != nil {
		t.Fatal("Init", err)
	}
	if m.DB == nil {
		t.Error("Init left no handle")
	}
	if err := m.Close(); err != nil {
		t.Errorf("Close: %v", err)
	}
	if err := (&DB{}).Close(); err != nil {
		t.Errorf("Close on an uninit host should be a no-op, got %v", err)
	}
}
