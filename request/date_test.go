package param

import (
	"testing"
	"time"
)

func TestDateFilterRange(t *testing.T) {
	year, month, day := time.Now().Date()

	cases := []struct {
		name     string
		filter   DateFilter
		wantDay  int
		wantMon  time.Month
		wantYear int
	}{
		{
			name:     "today truncates to midnight",
			filter:   DateFilter{Type: 1},
			wantDay:  day,
			wantMon:  month,
			wantYear: year,
		},
		{
			name:     "this month begins on the 1st",
			filter:   DateFilter{Type: 3},
			wantDay:  1,
			wantMon:  month,
			wantYear: year,
		},
		{
			name:     "this year begins on Jan 1",
			filter:   DateFilter{Type: 4},
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
		begin, _ := DateFilter{Type: 2}.Range(time.Local)
		if begin.Weekday() != time.Monday {
			t.Errorf("week begin weekday = %v, want Monday", begin.Weekday())
		}
		if begin.Hour() != 0 || begin.Minute() != 0 || begin.Second() != 0 {
			t.Errorf("week begin not truncated to midnight: %v", begin)
		}
	})

	t.Run("loc param is honored", func(t *testing.T) {
		begin, end := DateFilter{Type: 1}.Range(time.UTC)
		if begin.Location() != time.UTC {
			t.Errorf("begin location = %v, want UTC", begin.Location())
		}
		if end.Location() != time.UTC {
			t.Errorf("end location = %v, want UTC", end.Location())
		}
	})
}
