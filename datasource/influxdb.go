/*
 * Copyright 2026 hopeio. All rights reserved.
 * Licensed under the MIT License that can be found in the LICENSE file.
 * @Created by jyb
 */

package datasource

import (
	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
)

// InfluxDBConfig configures an InfluxDB v2 client.
type InfluxDBConfig struct {
	ServerURL string
	AuthToken string
}

// BeforeInject is a no-op satisfying the config injection contract.
func (c *InfluxDBConfig) BeforeInject() {}

// AfterInject is a no-op satisfying the config injection contract.
func (c *InfluxDBConfig) AfterInject() {}

// Build creates an InfluxDB v2 client. It performs no network I/O; the first
// request happens on the first API call.
func (c *InfluxDBConfig) Build() influxdb2.Client {
	return influxdb2.NewClient(c.ServerURL, c.AuthToken)
}

// InfluxDBClient is the injectable host around an InfluxDB v2 client.
type InfluxDBClient struct {
	Client influxdb2.Client
	Conf   InfluxDBConfig
}

// Config returns a pointer to the embedded Conf so injection writes into it.
func (m *InfluxDBClient) Config() any {
	return &m.Conf
}

// Init creates the InfluxDB client and stores it.
func (m *InfluxDBClient) Init() error {
	m.Client = m.Conf.Build()
	return nil
}

// Close shuts down the InfluxDB client if it was initialized.
func (m *InfluxDBClient) Close() error {
	if m.Client != nil {
		m.Client.Close()
	}
	return nil
}
