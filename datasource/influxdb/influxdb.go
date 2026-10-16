/*
 * Copyright 2026 hopeio. All rights reserved.
 * Licensed under the MIT License that can be found in the LICENSE file.
 * @Created by jyb
 */

package influxdb

import (
	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
)

// Config configures an InfluxDB v2 client.
type Config struct {
	ServerURL string
	AuthToken string
}

// BeforeInject is a no-op satisfying the config injection contract.
func (c *Config) BeforeInject() {}

// AfterInject is a no-op satisfying the config injection contract.
func (c *Config) AfterInject() {}

// Build creates an InfluxDB v2 client. It performs no network I/O; the first
// request happens on the first API call.
func (c *Config) Build() influxdb2.Client {
	return influxdb2.NewClient(c.ServerURL, c.AuthToken)
}

// Client is the injectable host around an InfluxDB v2 client.
type Client struct {
	Client influxdb2.Client
	Conf   Config
}

// Config returns a pointer to the embedded Conf so injection writes into it.
func (m *Client) Config() any {
	return &m.Conf
}

// Init creates the InfluxDB client and stores it.
func (m *Client) Init() error {
	m.Client = m.Conf.Build()
	return nil
}

// Close shuts down the InfluxDB client if it was initialized.
func (m *Client) Close() error {
	if m.Client != nil {
		m.Client.Close()
	}
	return nil
}
