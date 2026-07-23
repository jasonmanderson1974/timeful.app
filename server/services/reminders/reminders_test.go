package reminders

import (
	"strings"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"sirtom/server/models"
)

func TestIsReminderDue(t *testing.T) {
	now := time.Date(2026, 7, 23, 12, 0, 0, 0, time.UTC)

	cases := []struct {
		name          string
		start         time.Time
		leadTimeHours int
		want          bool
	}{
		{"before window", now.Add(48 * time.Hour), 24, false}, // dueAt = start-24h = +24h from now
		{"exactly at dueAt", now.Add(24 * time.Hour), 24, true},
		{"inside window", now.Add(2 * time.Hour), 24, true},
		{"start in the past", now.Add(-1 * time.Hour), 24, false},
		{"start is now", now, 24, false},
		// Degenerate lead=0 (clamp forbids it in practice): dueAt == start, and the
		// future-guard means now < start, so it never reads as due.
		{"zero lead, future start", now.Add(1 * time.Hour), 0, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := isReminderDue(now, tc.start, tc.leadTimeHours); got != tc.want {
				t.Errorf("isReminderDue(now, %v, %d) = %v, want %v", tc.start, tc.leadTimeHours, got, tc.want)
			}
		})
	}
}

func TestCollectRecipientEmails(t *testing.T) {
	userA := primitive.NewObjectID().Hex()
	userB := primitive.NewObjectID().Hex()

	lookup := func(userId string) string {
		switch userId {
		case userA:
			return "alice@example.com"
		case userB:
			return "" // signed-in user with no email on file — skipped
		default:
			return ""
		}
	}

	responses := []models.EventResponse{
		{UserId: "Guest Greg", Response: &models.Response{Email: "greg@example.com"}},  // guest email
		{UserId: userA, Response: &models.Response{}},                                  // signed-in, resolves via lookup
		{UserId: userB, Response: &models.Response{}},                                  // signed-in, no email -> skipped
		{UserId: "Guest Noemail", Response: &models.Response{}},                        // guest, no email, non-ObjectId -> skipped
		{UserId: userA, Response: &models.Response{}},                                  // duplicate of alice -> deduped
		{UserId: "Guest Greg2", Response: &models.Response{Email: "greg@example.com"}}, // duplicate guest email -> deduped
	}

	got := collectRecipientEmails(responses, lookup)
	want := []string{"greg@example.com", "alice@example.com"}

	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("index %d: got %q, want %q (full: %v)", i, got[i], want[i], got)
		}
	}
}

func TestCollectRsvpRecipientEmails(t *testing.T) {
	userGoing := primitive.NewObjectID().Hex()
	userNoEmail := primitive.NewObjectID().Hex()

	lookup := func(userId string) string {
		if userId == userNoEmail {
			return "signedin@example.com"
		}
		return ""
	}

	event := &models.Event{
		Rsvps: map[string]*models.Rsvp{
			"Guest Going":   {Status: models.RsvpGoing, Email: "going@example.com"},
			"Guest Maybe":   {Status: models.RsvpMaybe, Email: "maybe@example.com"},
			"Guest No":      {Status: models.RsvpNo, Email: "no@example.com"}, // excluded
			userGoing:       {Status: models.RsvpGoing, Email: "user-going@example.com"},
			userNoEmail:     {Status: models.RsvpMaybe},                             // resolved via lookup(key)
			"Guest Dup":     {Status: models.RsvpGoing, Email: "going@example.com"}, // dedup
			"Guest NilSkip": nil,
		},
	}

	got := collectRsvpRecipientEmails(event, lookup)

	want := map[string]bool{
		"going@example.com":      true,
		"maybe@example.com":      true,
		"user-going@example.com": true,
		"signedin@example.com":   true,
	}
	if len(got) != len(want) {
		t.Fatalf("got %v (len %d), want %d unique addrs", got, len(got), len(want))
	}
	for _, e := range got {
		if !want[e] {
			t.Errorf("unexpected recipient %q (decliner or dup leaked?): %v", e, got)
		}
	}
}

func TestBuildReminderEmail(t *testing.T) {
	desc := "Bring cigars."
	event := &models.Event{
		Name:        "Saturday Gathering",
		Description: &desc,
		GatheringReminder: &models.GatheringReminder{
			Timezone: "America/Los_Angeles",
		},
	}
	// 2026-08-01 02:00 UTC == 2026-07-31 19:00 PDT
	start := time.Date(2026, 8, 1, 2, 0, 0, 0, time.UTC)

	subject, body := buildReminderEmail(event, start)

	if !strings.Contains(subject, "Saturday Gathering") {
		t.Errorf("subject missing event name: %q", subject)
	}
	if !strings.Contains(body, "7:00 PM") {
		t.Errorf("body did not format start time in event timezone (expected 7:00 PM PDT):\n%s", body)
	}
	if !strings.Contains(body, "Friday, July 31") {
		t.Errorf("body did not format the local date:\n%s", body)
	}
	if !strings.Contains(body, desc) {
		t.Errorf("body missing description")
	}
}

func TestBuildReminderEmailTimezoneFallback(t *testing.T) {
	event := &models.Event{
		Name:              "No TZ Gathering",
		GatheringReminder: &models.GatheringReminder{}, // no timezone -> UTC
	}
	start := time.Date(2026, 8, 1, 2, 0, 0, 0, time.UTC)

	_, body := buildReminderEmail(event, start)
	if !strings.Contains(body, "2:00 AM UTC") {
		t.Errorf("body did not fall back to UTC formatting:\n%s", body)
	}
}
