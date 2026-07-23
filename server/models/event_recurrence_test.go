package models

import (
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestGatheringRecurrence_IsRecurring(t *testing.T) {
	cases := []struct {
		rec  *GatheringRecurrence
		want bool
	}{
		{nil, false},
		{&GatheringRecurrence{Frequency: RecurrenceNone}, false},
		{&GatheringRecurrence{Frequency: "bogus"}, false},
		{&GatheringRecurrence{Frequency: RecurrenceWeekly}, true},
		{&GatheringRecurrence{Frequency: RecurrenceBiweekly}, true},
		{&GatheringRecurrence{Frequency: RecurrenceMonthly}, true},
	}
	for _, c := range cases {
		if got := c.rec.IsRecurring(); got != c.want {
			t.Errorf("IsRecurring(%+v) = %v, want %v", c.rec, got, c.want)
		}
	}
}

func TestGatheringRecurrence_Step(t *testing.T) {
	// 7:00 PM local anchor, preserves time-of-day.
	anchor := time.Date(2026, 1, 15, 19, 0, 0, 0, time.UTC)
	cases := []struct {
		freq RecurrenceFrequency
		want time.Time
	}{
		{RecurrenceWeekly, time.Date(2026, 1, 22, 19, 0, 0, 0, time.UTC)},
		{RecurrenceBiweekly, time.Date(2026, 1, 29, 19, 0, 0, 0, time.UTC)},
		{RecurrenceMonthly, time.Date(2026, 2, 15, 19, 0, 0, 0, time.UTC)},
	}
	for _, c := range cases {
		r := &GatheringRecurrence{Frequency: c.freq}
		if got := r.Step(anchor); !got.Equal(c.want) {
			t.Errorf("Step(%s, %s) = %s, want %s", c.freq, anchor, got, c.want)
		}
	}
}

func TestGatheringRecurrence_MonthlyClamp(t *testing.T) {
	r := &GatheringRecurrence{Frequency: RecurrenceMonthly}

	// Jan 31 + 1 month must clamp to Feb 28 (2026 is not a leap year), NOT roll
	// into March the way time.AddDate would.
	jan31 := time.Date(2026, 1, 31, 8, 30, 0, 0, time.UTC)
	if got, want := r.Step(jan31), time.Date(2026, 2, 28, 8, 30, 0, 0, time.UTC); !got.Equal(want) {
		t.Errorf("Jan 31 + 1mo = %s, want %s", got, want)
	}

	// Leap year: Jan 31 2028 -> Feb 29.
	jan31leap := time.Date(2028, 1, 31, 8, 30, 0, 0, time.UTC)
	if got, want := r.Step(jan31leap), time.Date(2028, 2, 29, 8, 30, 0, 0, time.UTC); !got.Equal(want) {
		t.Errorf("Jan 31 2028 + 1mo = %s, want %s", got, want)
	}

	// Year boundary: Dec 15 -> Jan 15 next year.
	dec15 := time.Date(2026, 12, 15, 12, 0, 0, 0, time.UTC)
	if got, want := r.Step(dec15), time.Date(2027, 1, 15, 12, 0, 0, 0, time.UTC); !got.Equal(want) {
		t.Errorf("Dec 15 + 1mo = %s, want %s", got, want)
	}
}

func TestGatheringRecurrence_NextOccurrenceAfter(t *testing.T) {
	weekly := &GatheringRecurrence{Frequency: RecurrenceWeekly}
	start := time.Date(2026, 1, 1, 19, 0, 0, 0, time.UTC) // Thu

	// `after` just past the start -> the very next weekly occurrence.
	after := start.Add(time.Hour)
	if got, want := weekly.NextOccurrenceAfter(start, after), start.AddDate(0, 0, 7); !got.Equal(want) {
		t.Errorf("next after +1h = %s, want %s", got, want)
	}

	// Skip a long outage: `after` is ~5 weeks out -> jump straight to the next
	// future occurrence, not replay the intervening ones.
	after = start.AddDate(0, 0, 30) // 30 days later
	got := weekly.NextOccurrenceAfter(start, after)
	if !got.After(after) {
		t.Fatalf("occurrence %s is not after %s", got, after)
	}
	if got.Sub(after) > 7*24*time.Hour {
		t.Errorf("occurrence %s is more than one interval after %s (didn't skip)", got, after)
	}
	// Specifically day 1 + 35 = Feb 5 (5 weeks).
	if want := time.Date(2026, 2, 5, 19, 0, 0, 0, time.UTC); !got.Equal(want) {
		t.Errorf("next after 30d = %s, want %s", got, want)
	}

	// Non-recurring -> zero time.
	none := &GatheringRecurrence{Frequency: RecurrenceNone}
	if got := none.NextOccurrenceAfter(start, after); !got.IsZero() {
		t.Errorf("non-recurring should return zero time, got %s", got)
	}
}

func TestGatheringRecurrence_RRULE(t *testing.T) {
	until := primitive.NewDateTimeFromTime(time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC))
	cases := []struct {
		rec  *GatheringRecurrence
		want string
	}{
		{nil, ""},
		{&GatheringRecurrence{Frequency: RecurrenceNone}, ""},
		{&GatheringRecurrence{Frequency: RecurrenceWeekly}, "FREQ=WEEKLY"},
		{&GatheringRecurrence{Frequency: RecurrenceBiweekly}, "FREQ=WEEKLY;INTERVAL=2"},
		{&GatheringRecurrence{Frequency: RecurrenceMonthly}, "FREQ=MONTHLY"},
		{&GatheringRecurrence{Frequency: RecurrenceMonthly, Until: &until}, "FREQ=MONTHLY;UNTIL=20261231T000000Z"},
	}
	for _, c := range cases {
		if got := c.rec.RRULE(); got != c.want {
			t.Errorf("RRULE(%+v) = %q, want %q", c.rec, got, c.want)
		}
	}
}
