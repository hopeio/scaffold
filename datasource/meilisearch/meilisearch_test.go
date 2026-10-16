/*
 * Copyright 2026 hopeio. All rights reserved.
 * Licensed under the MIT License that can be found in the LICENSE file.
 * @Created by jyb
 */

package meilisearch

import "testing"

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
	_ hostContract = (*Client)(nil)
	_ configHooks  = (*Config)(nil)
)

// Build must not touch the network, so an unreachable endpoint still inits.
func TestInitDoesNotDial(t *testing.T) {
	m := &Client{Conf: Config{Host: "http://127.0.0.1:1", APIKey: "k"}}
	if err := m.Init(); err != nil {
		t.Fatalf("Init: %v", err)
	}
	if m.Service == nil {
		t.Error("Init left no service")
	}
	if err := m.Close(); err != nil {
		t.Errorf("Close: %v", err)
	}
}
