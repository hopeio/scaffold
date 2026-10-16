/*
 * Copyright 2024 hopeio. All rights reserved.
 * Licensed under the MIT License that can be found in the LICENSE file.
 * @Created by jyb
 */

package esv8

// Response shapes of the Elasticsearch 8 _search API. They live here rather
// than in gox because the field set ("_shards", hits.total.value, "_source") is
// version-specific: v7 differs, so these types belong next to the v8 client
// helpers that decode them.

// SearchResponse is one _search response with hits decoded into T.
type SearchResponse[T any] struct {
	Took     int     `json:"took"`
	TimedOut bool    `json:"timed_out"`
	Shards   Shards  `json:"_shards"`
	Hits     Hits[T] `json:"hits"`
}

type Shards struct {
	Total      int `json:"total"`
	Successful int `json:"successful"`
	Skipped    int `json:"skipped"`
	Failed     int `json:"failed"`
}

type Hits[T any] struct {
	Total    Total    `json:"total"`
	MaxScore any      `json:"max_score"`
	Hits     []Hit[T] `json:"hits"`
}

type Hit[T any] struct {
	Index  string      `json:"_index"`
	Id     string      `json:"_id"`
	Score  interface{} `json:"_score"`
	Source T           `json:"_source"`
	Sort   []int       `json:"sort"`
}

type Total struct {
	Value    int    `json:"value"`
	Relation string `json:"relation"`
}
