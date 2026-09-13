/*
 * Copyright 2024 hopeio. All rights reserved.
 * Licensed under the MIT License that can be found in the LICENSE file.
 * @Created by jyb
 */

package param

import (
	"time"
)

// DateFilterType 时间筛选类型
type DateFilterType int32

const (
	DateFilterNone       DateFilterType = 0 // 不限
	DateFilterToday      DateFilterType = 1 // 今天
	DateFilterThisWeek   DateFilterType = 2 // 本周（周一为起点）
	DateFilterThisMonth  DateFilterType = 3 // 本月
	DateFilterThisYear   DateFilterType = 4 // 今年
	DateFilterLast7Days  DateFilterType = 5 // 近7天
	DateFilterLast30Days DateFilterType = 6 // 近30天
	DateFilterLast90Days DateFilterType = 7 // 近90天
	DateFilterLastYear   DateFilterType = 8 // 近一年
)

// Range 根据筛选类型解析区间起止时间，loc 指定时区。结束时间恒为当前时刻。
// 起始时间截断到当日 00:00:00；DateFilterNone 返回零值 begin（表示不限下限）。
func (t DateFilterType) Range(loc *time.Location) (time.Time, time.Time) {
	now := time.Now().In(loc)
	if t == DateFilterNone {
		return time.Time{}, now
	}
	year, month, day := now.Date()
	var begin time.Time
	switch t {
	case DateFilterToday:
		begin = time.Date(year, month, day, 0, 0, 0, 0, loc)
	case DateFilterThisWeek:
		weekday := now.Weekday()
		if weekday == time.Sunday {
			weekday = 6
		} else {
			weekday -= 1
		}
		ws := now.AddDate(0, 0, -int(weekday))
		begin = time.Date(ws.Year(), ws.Month(), ws.Day(), 0, 0, 0, 0, loc)
	case DateFilterThisMonth:
		begin = time.Date(year, month, 1, 0, 0, 0, 0, loc)
	case DateFilterThisYear:
		begin = time.Date(year, 1, 1, 0, 0, 0, 0, loc)
	case DateFilterLast7Days:
		begin = dayStart(now.AddDate(0, 0, -7), loc)
	case DateFilterLast30Days:
		begin = dayStart(now.AddDate(0, 0, -30), loc)
	case DateFilterLast90Days:
		begin = dayStart(now.AddDate(0, 0, -90), loc)
	case DateFilterLastYear:
		begin = dayStart(now.AddDate(-1, 0, 0), loc)
	}
	return begin, now
}

func dayStart(t time.Time, loc *time.Location) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, loc)
}
