/*
 * Copyright 2024 hopeio. All rights reserved.
 * Licensed under the MIT License that can be found in the LICENSE file.
 * @Created by jyb
 */

package database

import "strconv"

// TableName returns a sharded table name based on the numeric id.
// IDs below 200,000,000 use the base name; higher IDs get a numeric suffix for
// horizontal sharding (id/2e8: 1, 2, ... 10, 11 ...).
// 曾用 string(byte(n+49)) 拼后缀：n≥10 时产生 ';' '<' 等乱码字符（';' 还是 SQL 危险字符）。
func TableName(name string, id uint64) string {
	if id < 2000_00000 {
		return name
	}
	return name + "_" + strconv.FormatUint(id/2000_00000, 10)
}
