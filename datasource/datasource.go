/*
 * Copyright 2026 hopeio. All rights reserved.
 * Licensed under the MIT License that can be found in the LICENSE file.
 * @Created by jyb
 */

// Package datasource holds optional external data-source clients used by
// application dao layers: InfluxDB (time series), FlightSQL (Arrow SQL over
// ADBC), DuckDB (embedded analytics) and MeiliSearch (search).
//
// Each one follows the same duck-typed injection contract that the bootstrap
// layer expects, without importing it:
//
//	FooConfig.BeforeInject() / AfterInject()  no-ops, config-side hooks
//	FooConfig.Build()                         construct the client / open the handle
//	Foo.Config() any                          pointer to the embedded Conf
//	Foo.Init() error                          build and store it
//	Foo.Close() error                         release it if it was initialized
//
// A binary injects a pair like:
//
//	Duck  datasource.DuckDB           `init:"config:duckdb"`
//	Meili datasource.MeiliSearchClient `init:"config:meilisearch"`
//
// Trade-off to keep in mind: everything in this package is linked together, so
// importing it pulls DuckDB (CGO plus per-platform static libraries) and Apache
// Arrow (via the FlightSQL ADBC driver) into the binary even if only one client
// is used. Binaries that are sensitive to that should not import this package
// from a hot path.
package datasource
