package reminders

import (
	"testing"

	"sirtom/server/models"
)

func TestChronicleAttendees(t *testing.T) {
	rsvps := map[string]*models.Rsvp{
		"u1":  {Status: models.RsvpGoing, Name: "Alice", GuestCount: 2}, // 3 heads
		"Bob": {Status: models.RsvpMaybe, Name: "Bob"},                  // 1 head
		"u3":  {Status: models.RsvpNo, Name: "Carol"},                   // decliner — excluded
		"u4":  {Status: models.RsvpGoing},                               // no name -> key used
		"u5":  nil,                                                      // nil — skipped
	}

	attendees, headCount := chronicleAttendees(rsvps)

	if len(attendees) != 3 {
		t.Fatalf("expected 3 attendees (going/maybe), got %d: %+v", len(attendees), attendees)
	}
	if headCount != 5 { // (1+2) + 1 + 1
		t.Errorf("expected headCount 5, got %d", headCount)
	}

	names := map[string]bool{}
	for _, a := range attendees {
		names[a.Name] = true
		if a.Status == models.RsvpNo {
			t.Errorf("decliner leaked into attendees: %+v", a)
		}
	}
	if names["Carol"] {
		t.Error("Carol (declined) should not be an attendee")
	}
	if !names["u4"] {
		t.Error("expected nameless RSVP to fall back to its key (u4)")
	}
}

func TestChronicleAttendees_Empty(t *testing.T) {
	attendees, headCount := chronicleAttendees(nil)
	if len(attendees) != 0 || headCount != 0 {
		t.Errorf("nil rsvps should give empty roster, got %d attendees / head %d", len(attendees), headCount)
	}
}
