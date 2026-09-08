package param

import (
	"testing"
	"time"
)

func TestDateFilterRange(t *testing.T) {
	now := time.Date(2026, time.September, 16, 15, 30, 45, 0, time.Local)

	// Helper to freeze time.Now via a shim is not available; assert against real now
	// using the same Date() extraction the implementation uses.
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
			begin, end := tc.filter.Range()
			if begin.Year() != tc.wantYear || begin.Month() != tc.wantMon || begin.Day() != tc.wantDay {
				t.Errorf("begin = %v, want %d-%02d-%02d", begin, tc.wantYear, tc.wantMon, tc.wantDay)
			}
			if begin.Hour() != 0 || begin.Minute() != 0 || begin.Second() != 0 {
				t.Errorf("begin not truncated to midnight: %v", begin)
			}
			if end.IsZero() {
				t.Errorf("end should default to now, got zero")
			}
			_ = now
		})
	}

	t.Run("explicit begin kept when end missing", func(t *testing.T) {
		explicit := time.Date(2026, 1, 2, 3, 4, 5, 0, time.Local)
		f := DateFilter{Begin: explicit}
		begin, end := f.Range()
		if !begin.Equal(explicit) {
			t.Errorf("begin = %v, want explicit %v", begin, explicit)
		}
		if end.IsZero() {
			t.Errorf("end should default to now")
		}
	})

	t.Run("explicit end kept when begin missing", func(t *testing.T) {
		explicit := time.Date(2026, 1, 2, 3, 4, 5, 0, time.Local)
		f := DateFilter{End: explicit}
		begin, end := f.Range()
		if !end.Equal(explicit) {
			t.Errorf("end = %v, want explicit %v", end, explicit)
		}
		if !begin.IsZero() {
			t.Errorf("begin should be open (zero) when missing, got %v", begin)
		}
	})

	t.Run("both explicit returned as-is", func(t *testing.T) {
		b := time.Date(2026, 1, 1, 0, 0, 0, 0, time.Local)
		e := time.Date(2026, 2, 2, 0, 0, 0, 0, time.Local)
		f := DateFilter{Begin: b, End: e}
		begin, end := f.Range()
		if !begin.Equal(b) || !end.Equal(e) {
			t.Errorf("got (%v,%v), want (%v,%v)", begin, end, b, e)
		}
	})
}
