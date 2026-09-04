package main

import (
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

func TestOldestAgeUsesAuthoritativeTimestampAndClampsEmptyOrFuture(t *testing.T) {
	now := time.Date(2026, time.September, 5, 2, 0, 0, 0, time.UTC)

	cases := []struct {
		name   string
		oldest pgtype.Timestamptz
		want   time.Duration
	}{
		{name: "empty", oldest: pgtype.Timestamptz{}, want: 0},
		{name: "ninety seconds old", oldest: pgtype.Timestamptz{Time: now.Add(-90 * time.Second), Valid: true}, want: 90 * time.Second},
		{name: "future timestamp clamps to zero", oldest: pgtype.Timestamptz{Time: now.Add(time.Second), Valid: true}, want: 0},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := oldestAge(now, tc.oldest); got != tc.want {
				t.Fatalf("oldestAge() = %s, want %s", got, tc.want)
			}
		})
	}
}
