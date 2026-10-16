/*
 * Copyright 2026 hopeio. All rights reserved.
 * Licensed under the MIT License that can be found in the LICENSE file.
 * @Created by jyb
 */

package meilisearch

import (
	meili "github.com/meilisearch/meilisearch-go"
)

// Config points at a MeiliSearch instance.
type Config struct {
	Host   string
	APIKey string
}

// BeforeInject is a no-op satisfying the config injection contract.
func (c *Config) BeforeInject() {}

// AfterInject is a no-op satisfying the config injection contract.
func (c *Config) AfterInject() {}

// Build creates the ServiceManager. Nothing is sent until the first API call, so
// injecting this never blocks startup on an unreachable host.
//
// The SDK is imported as `meili` because this package shares its name: inside
// `package meilisearch` a second identifier spelled the same way is unreadable.
func (c *Config) Build() meili.ServiceManager {
	return meili.New(c.Host, meili.WithAPIKey(c.APIKey))
}

// Client is the injectable host around a MeiliSearch client.
type Client struct {
	Service meili.ServiceManager
	Conf    Config
}

// Config returns a pointer to the embedded Conf so injection writes into it.
func (m *Client) Config() any {
	return &m.Conf
}

// Init creates the MeiliSearch client and stores it.
func (m *Client) Init() error {
	m.Service = m.Conf.Build()
	return nil
}

// Close has nothing to release: the client rides on a shared net/http transport
// with no long-lived handle. It exists to satisfy the host contract.
func (m *Client) Close() error {
	return nil
}
