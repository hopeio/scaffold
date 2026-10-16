/*
 * Copyright 2026 hopeio. All rights reserved.
 * Licensed under the MIT License that can be found in the LICENSE file.
 * @Created by jyb
 */

package flightsql

import (
	"database/sql"
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

// The blank driver import is the whole point of this package: dropping it would
// only surface at runtime as "unknown driver".
func TestDriverRegistered(t *testing.T) {
	if !strings.Contains(strings.Join(sql.Drivers(), ","), "flightsql") {
		t.Error("flightsql driver not registered by the blank import")
	}
}

func TestUninitCloseIsNoop(t *testing.T) {
	if err := (&DB{}).Close(); err != nil {
		t.Errorf("got %v, want nil", err)
	}
}
