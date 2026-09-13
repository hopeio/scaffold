package param

import (
	"testing"
	"time"
)

func TestDateFilterTypeRange(t *testing.T) {
	year, month, day := time.Now().Date()

	cases := []struct {
		name     string
		filter   DateFilterType
		wantDay  int
		wantMon  time.Month
		wantYear int
	}{
		{
			name:     "today truncates to midnight",
			filter:   DateFilterToday,
			wantDay:  day,
			wantMon:  month,
			wantYear: year,
		},
		{
			name:     "this month begins on the 1st",
			filter:   DateFilterThisMonth,
			wantDay:  1,
			wantMon:  month,
			wantYear: year,
		},
		{
			name:     "this year begins on Jan 1",
			filter:   DateFilterThisYear,
			wantDay:  1,
			wantMon:  time.January,
			wantYear: year,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			begin, end := tc.filter.Range(time.Local)
			if begin.Year() != tc.wantYear || begin.Month() != tc.wantMon || begin.Day() != tc.wantDay {
				t.Errorf("begin = %v, want %d-%02d-%02d", begin, tc.wantYear, tc.wantMon, tc.wantDay)
			}
			if begin.Hour() != 0 || begin.Minute() != 0 || begin.Second() != 0 {
				t.Errorf("begin not truncated to midnight: %v", begin)
			}
			if end.IsZero() {
				t.Errorf("end should default to now, got zero")
			}
		})
	}

	t.Run("this week begins on Monday midnight", func(t *testing.T) {
		begin, _ := DateFilterThisWeek.Range(time.Local)
		if begin.Weekday() != time.Monday {
			t.Errorf("week begin weekday = %v, want Monday", begin.Weekday())
		}
		if begin.Hour() != 0 || begin.Minute() != 0 || begin.Second() != 0 {
			t.Errorf("week begin not truncated to midnight: %v", begin)
		}
	})

	t.Run("last N days begin is N days ago at midnight", func(t *testing.T) {
		for _, dur := range []struct {
			t     DateFilterType
			days  int
		}{
			{DateFilterLast7Days, 7},
			{DateFilterLast30Days, 30},
			{DateFilterLast90Days, 90},
		} {
			begin, _ := dur.t.Range(time.Local)
			got := time.Now().Sub(begin).Hours() / 24
			if got < float64(dur.days-1) || got > float64(dur.days+1) {
				t.Errorf("%v: begin %v ~= %d days ago (got %.1f)", dur.t, begin, dur.days, got)
			}
			if begin.Hour() != 0 || begin.Minute() != 0 || begin.Second() != 0 {
				t.Errorf("%v: begin not truncated to midnight: %v", dur.t, begin)
			}
		}
	})

	t.Run("none returns zero begin", func(t *testing.T) {
		begin, _ := DateFilterNone.Range(time.Local)
		if !begin.IsZero() {
			t.Errorf("none begin = %v, want zero", begin)
		}
	})

	t.Run("loc param is honored", func(t *testing.T) {
		begin, end := DateFilterToday.Range(time.UTC)
		if begin.Location() != time.UTC {
			t.Errorf("begin location = %v, want UTC", begin.Location())
		}
		if end.Location() != time.UTC {
			t.Errorf("end location = %v, want UTC", end.Location())
		}
	})
}
