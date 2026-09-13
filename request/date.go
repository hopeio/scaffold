/*
 * Copyright 2024 hopeio. All rights reserved.
 * Licensed under the MIT License that can be found in the LICENSE file.
 * @Created by jyb
 */

package param

import (
	"time"
)

// DateFilterType 时间筛选类型：1-今天, 2-本周，3-本月，4-今年
type DateFilterType int32

type DateFilter struct {
	Type DateFilterType `json:"type" comment:"1-今天, 2-本周，3-本月，4-今年"`
}

// Range 根据 Type 解析区间起止时间，loc 指定时区。
// 1=今天 2=本周 3=本月 4=今年；起始时间截断到当日 00:00:00。结束时间为当前时刻。
func (d DateFilter) Range(loc *time.Location) (time.Time, time.Time) {
	now := time.Now().In(loc)
	year, month, day := now.Date()
	var begin time.Time
	switch d.Type {
	case 1:
		begin = time.Date(year, month, day, 0, 0, 0, 0, loc)
	case 2:
		weekday := now.Weekday()
		if weekday == time.Sunday {
			weekday = 6
		} else {
			weekday -= 1
		}
		ws := now.AddDate(0, 0, -int(weekday))
		begin = time.Date(ws.Year(), ws.Month(), ws.Day(), 0, 0, 0, 0, loc)
	case 3:
		begin = time.Date(year, month, 1, 0, 0, 0, 0, loc)
	case 4:
		begin = time.Date(year, 1, 1, 0, 0, 0, 0, loc)
	}
	return begin, now
}
