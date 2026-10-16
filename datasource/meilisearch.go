/*
 * Copyright 2026 hopeio. All rights reserved.
 * Licensed under the MIT License that can be found in the LICENSE file.
 * @Created by jyb
 */

package datasource

import (
	"github.com/meilisearch/meilisearch-go"
)

// MeiliSearchConfig points at a MeiliSearch instance.
type MeiliSearchConfig struct {
	Host   string
	APIKey string
}

// BeforeInject is a no-op satisfying the config injection contract.
func (c *MeiliSearchConfig) BeforeInject() {}

// AfterInject is a no-op satisfying the config injection contract.
func (c *MeiliSearchConfig) AfterInject() {}

// Build creates the ServiceManager. Nothing is sent until the first API call, so
// injecting this never blocks startup on an unreachable host.
func (c *MeiliSearchConfig) Build() meilisearch.ServiceManager {
	return meilisearch.New(c.Host, meilisearch.WithAPIKey(c.APIKey))
}

// MeiliSearchClient is the injectable host around a MeiliSearch client.
type MeiliSearchClient struct {
	Service meilisearch.ServiceManager
	Conf    MeiliSearchConfig
}

// Config returns a pointer to the embedded Conf so injection writes into it.
func (m *MeiliSearchClient) Config() any {
	return &m.Conf
}

// Init creates the MeiliSearch client and stores it.
func (m *MeiliSearchClient) Init() error {
	m.Service = m.Conf.Build()
	return nil
}

// Close has nothing to release: the client rides on a shared net/http transport
// with no long-lived handle. It exists to satisfy the host contract.
func (m *MeiliSearchClient) Close() error {
	return nil
}
