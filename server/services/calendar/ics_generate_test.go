package calendar

import (
	"strings"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"schej.it/server/models"
)

func TestGenerateEventICS(t *testing.T) {
	desc := "Bring cigars."
	venue := "The Lodge, 123 Main St"
	start := time.Date(2026, 8, 1, 2, 0, 0, 0, time.UTC)
	end := start.Add(2 * time.Hour)

	event := &models.Event{
		Id:          primitive.NewObjectID(),
		Name:        "Saturday Gathering",
		Description: &desc,
		Location:    &venue,
		ScheduledEvent: &models.CalendarEvent{
			Summary:   "Saturday Gathering",
			StartDate: primitive.NewDateTimeFromTime(start),
			EndDate:   primitive.NewDateTimeFromTime(end),
		},
	}

	out, err := GenerateEventICS(event)
	if err != nil {
		t.Fatalf("GenerateEventICS: %v", err)
	}
	s := string(out)

	mustContain := []string{
		"BEGIN:VCALENDAR",
		"VERSION:2.0",
		"BEGIN:VEVENT",
		"SUMMARY:Saturday Gathering",
		"DTSTART:20260801T020000Z",
		"DTEND:20260801T040000Z",
		"STATUS:CONFIRMED",
		"LOCATION:The Lodge\\, 123 Main St",
		"UID:" + event.Id.Hex() + "@timeful.app",
		"END:VEVENT",
		"END:VCALENDAR",
	}
	for _, want := range mustContain {
		if !strings.Contains(s, want) {
			t.Errorf("ICS missing %q:\n%s", want, s)
		}
	}
}

func TestGenerateEventICS_NoScheduledEvent(t *testing.T) {
	event := &models.Event{Name: "Unscheduled"}
	if _, err := GenerateEventICS(event); err == nil {
		t.Fatal("expected error when event has no confirmed gathering, got nil")
	}
	if _, err := GenerateEventICS(nil); err == nil {
		t.Fatal("expected error for nil event, got nil")
	}
}
