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

// Range resolves Begin/End from explicit values or the Type preset (1=today, 2=this week, 3=this month, 4=this year).
// An explicitly set Begin/End is kept; only the missing side falls back to the preset. Preset begins are
// truncated to midnight of the day they land on.
func (d *DateFilter) Range() (time.Time, time.Time) {
	var zero time.Time
	if d.Begin != zero && d.End != zero {
		return d.Begin, d.End
	}
	now := time.Now()
	year, month, day := now.Date()
	end := d.End
	if end == zero {
		end = now
	}
	begin := d.Begin
	if begin == zero {
		switch d.Type {
		case 1:
			begin = time.Date(year, month, day, 0, 0, 0, 0, time.Local)
		case 2:
			weekday := now.Weekday()
			if weekday == time.Sunday {
				weekday = 6
			} else {
				weekday -= 1
			}
			ws := now.AddDate(0, 0, -int(weekday))
			begin = time.Date(ws.Year(), ws.Month(), ws.Day(), 0, 0, 0, 0, time.Local)
		case 3:
			begin = time.Date(year, month, 1, 0, 0, 0, 0, time.Local)
		case 4:
			begin = time.Date(year, 1, 1, 0, 0, 0, 0, time.Local)
		}
	}
	d.Begin, d.End = begin, end
	return begin, end
}
