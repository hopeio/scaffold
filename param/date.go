/*
 * Copyright 2024 hopeio. All rights reserved.
 * Licensed under the MIT License that can be found in the LICENSE file.
 * @Created by jyb
 */

package param

import (
	"time"
)

type DateFilter struct {
	Begin time.Time `json:"begin" comment:"起始时间"`
	End   time.Time `json:"end" comment:"结束时间"`
	Type  int       `json:"type" comment:"1-今天, 2-本周，3-本月，4-今年"`
}

// 赋值本周期，并返回下周期日期
func (d *DateFilter) Scope() (time.Time, time.Time) {
	var zero time.Time
	if d.Begin != zero && d.End != zero {
		return d.Begin, d.End
	}
	//如果传的是RangeEnum，截止日期都是这一天
	now := time.Now()
	year, month, day := time.Now().Date()
	d.End = now
	switch d.Type {
	case 1:
		d.Begin = time.Date(year, month, day, 0, 0, 0, 0, time.Local)
	case 2:
		weekday := now.Weekday()
		if weekday == time.Sunday {
			weekday = 6
		} else {
			weekday -= 1
		}
		d.Begin = now.AddDate(0, 0, -int(weekday))
	case 3:
		d.Begin = time.Date(year, month, 0, 0, 0, 0, 0, time.Local)
	case 4:
		d.Begin = time.Date(year, 0, 0, 0, 0, 0, 0, time.Local)
	}
	return d.Begin, d.End
}
