package reminders

import (
	"context"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"sirtom/server/db"
	"sirtom/server/models"
)

func cleanupChronicle(eventId primitive.ObjectID) {
	ctx := context.Background()
	db.EventsCollection.DeleteOne(ctx, bson.M{"_id": eventId})
	db.ChronicleCollection.DeleteMany(ctx, bson.M{"eventId": eventId})
}

// A one-off gathering whose time has passed is captured into the Chronicle
// exactly once, and the event is flagged so it isn't re-captured.
func TestArchivePastGatherings_CapturesOnce(t *testing.T) {
	requireDB(t)
	ctx := context.Background()

	eventId := primitive.NewObjectID()
	now := time.Now()
	start := now.Add(-3 * time.Hour)
	venue := "The Lodge"

	_, err := db.EventsCollection.InsertOne(ctx, models.Event{
		Id:       eventId,
		Name:     "Past One-off",
		Type:     models.SPECIFIC_DATES,
		Location: &venue,
		ScheduledEvent: &models.CalendarEvent{
			StartDate: primitive.NewDateTimeFromTime(start),
			EndDate:   primitive.NewDateTimeFromTime(start.Add(2 * time.Hour)),
		},
		Rsvps: map[string]*models.Rsvp{
			"Alice": {Status: models.RsvpGoing, Name: "Alice", GuestCount: 1},
			"Bob":   {Status: models.RsvpNo, Name: "Bob"},
		},
	})
	if err != nil {
		t.Fatalf("insert event: %v", err)
	}
	defer cleanupChronicle(eventId)

	archivePastGatherings(now)

	// One entry captured, headcount 2 (Alice + 1 guest), Bob (declined) excluded.
	var entries []models.ChronicleEntry
	cur, _ := db.ChronicleCollection.Find(ctx, bson.M{"eventId": eventId})
	cur.All(ctx, &entries)
	if len(entries) != 1 {
		t.Fatalf("expected 1 chronicle entry, got %d", len(entries))
	}
	if entries[0].HeadCount != 2 {
		t.Errorf("expected headCount 2, got %d", entries[0].HeadCount)
	}
	if entries[0].Location == nil || *entries[0].Location != venue {
		t.Errorf("expected venue captured, got %v", entries[0].Location)
	}
	if len(entries[0].Attendees) != 1 {
		t.Errorf("expected only going/maybe attendee, got %+v", entries[0].Attendees)
	}

	// Event flagged chronicled.
	var reloaded models.Event
	db.EventsCollection.FindOne(ctx, bson.M{"_id": eventId}).Decode(&reloaded)
	if !reloaded.Chronicled {
		t.Error("expected event flagged chronicled")
	}

	// Second run must not add a duplicate.
	archivePastGatherings(now)
	count, _ := db.ChronicleCollection.CountDocuments(ctx, bson.M{"eventId": eventId})
	if count != 1 {
		t.Errorf("expected no duplicate on second run, got %d entries", count)
	}

	// The read path used by GET /chronicle surfaces the entry.
	listed, err := db.GetChronicleEntries(200)
	if err != nil {
		t.Fatalf("GetChronicleEntries: %v", err)
	}
	found := false
	for _, e := range listed {
		if e.EventId == eventId {
			found = true
			break
		}
	}
	if !found {
		t.Error("GetChronicleEntries did not return the captured gathering")
	}
}

// A future one-off gathering is not captured (it hasn't happened yet).
func TestArchivePastGatherings_SkipsFuture(t *testing.T) {
	requireDB(t)
	ctx := context.Background()

	eventId := primitive.NewObjectID()
	now := time.Now()
	start := now.Add(48 * time.Hour)

	_, err := db.EventsCollection.InsertOne(ctx, models.Event{
		Id:   eventId,
		Name: "Upcoming One-off",
		Type: models.SPECIFIC_DATES,
		ScheduledEvent: &models.CalendarEvent{
			StartDate: primitive.NewDateTimeFromTime(start),
			EndDate:   primitive.NewDateTimeFromTime(start.Add(2 * time.Hour)),
		},
	})
	if err != nil {
		t.Fatalf("insert event: %v", err)
	}
	defer cleanupChronicle(eventId)

	archivePastGatherings(now)

	count, _ := db.ChronicleCollection.CountDocuments(ctx, bson.M{"eventId": eventId})
	if count != 0 {
		t.Errorf("future gathering should not be chronicled, got %d entries", count)
	}
}

// When a recurring gathering rolls forward, its just-completed occurrence is
// captured into the Chronicle (with that occurrence's attendees) before the
// RSVPs are cleared.
func TestAdvanceRecurringGatherings_CapturesOccurrence(t *testing.T) {
	requireDB(t)
	ctx := context.Background()

	eventId := primitive.NewObjectID()
	now := time.Now()
	start := now.AddDate(0, 0, -8) // weekly occurrence ended a week+ ago
	startDT := primitive.NewDateTimeFromTime(start)

	_, err := db.EventsCollection.InsertOne(ctx, models.Event{
		Id:   eventId,
		Name: "Weekly Recurring",
		Type: models.SPECIFIC_DATES,
		ScheduledEvent: &models.CalendarEvent{
			StartDate: startDT,
			EndDate:   primitive.NewDateTimeFromTime(start.Add(2 * time.Hour)),
		},
		GatheringRecurrence: &models.GatheringRecurrence{Frequency: models.RecurrenceWeekly},
		Rsvps: map[string]*models.Rsvp{
			"Greg": {Status: models.RsvpGoing, Name: "Greg"},
		},
	})
	if err != nil {
		t.Fatalf("insert event: %v", err)
	}
	defer cleanupChronicle(eventId)

	advanceRecurringGatherings(now)

	// The completed occurrence (original startDate) is captured with Greg.
	var entry models.ChronicleEntry
	if err := db.ChronicleCollection.FindOne(ctx, bson.M{"eventId": eventId, "startDate": startDT}).Decode(&entry); err != nil {
		t.Fatalf("expected chronicle entry for the completed occurrence: %v", err)
	}
	if entry.HeadCount != 1 || len(entry.Attendees) != 1 || entry.Attendees[0].Name != "Greg" {
		t.Errorf("expected Greg captured (head 1), got head=%d attendees=%+v", entry.HeadCount, entry.Attendees)
	}

	// And the event itself rolled forward + cleared RSVPs (C5 behavior).
	var reloaded models.Event
	db.EventsCollection.FindOne(ctx, bson.M{"_id": eventId}).Decode(&reloaded)
	if !reloaded.ScheduledEvent.StartDate.Time().After(now) {
		t.Error("expected gathering advanced to a future occurrence")
	}
	if len(reloaded.Rsvps) != 0 {
		t.Error("expected RSVPs cleared after advance")
	}
}
