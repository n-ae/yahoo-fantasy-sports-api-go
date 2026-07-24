package service

import (
	"testing"
	"time"
)

func TestCurrentNBASeason(t *testing.T) {
	cases := []struct {
		date time.Time
		want string
	}{
		{time.Date(2025, time.October, 1, 0, 0, 0, 0, time.UTC), "2025-26"},
		{time.Date(2025, time.December, 31, 0, 0, 0, 0, time.UTC), "2025-26"},
		{time.Date(2026, time.January, 15, 0, 0, 0, 0, time.UTC), "2025-26"},
		{time.Date(2026, time.June, 30, 0, 0, 0, 0, time.UTC), "2025-26"},
		{time.Date(2026, time.July, 24, 0, 0, 0, 0, time.UTC), "2025-26"}, // offseason -> last season
		{time.Date(2026, time.October, 20, 0, 0, 0, 0, time.UTC), "2026-27"},
		{time.Date(2009, time.November, 1, 0, 0, 0, 0, time.UTC), "2009-10"}, // zero-padded year suffix
	}
	for _, c := range cases {
		if got := currentNBASeason(c.date); got != c.want {
			t.Errorf("currentNBASeason(%s) = %q, want %q", c.date.Format("2006-01"), got, c.want)
		}
	}
}
