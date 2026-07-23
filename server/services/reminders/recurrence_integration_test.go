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

// A recurring gathering whose occurrence has ended rolls forward to the next
// future occurrence, clearing the prior cycle's RSVPs and re-arming the reminder.
func TestAdvanceRecurringGatherings_RollsForward(t *testing.T) {
	requireDB(t)
	ctx := context.Background()

	eventId := primitive.NewObjectID()
	now := time.Now()
	// Weekly gathering whose last occurrence started 8 days ago (so it has ended).
	start := now.AddDate(0, 0, -8)
	startDT := primitive.NewDateTimeFromTime(start)
	sentAt := startDT

	_, err := db.EventsCollection.InsertOne(ctx, models.Event{
		Id:   eventId,
		Name: "Weekly Poker",
		Type: models.SPECIFIC_DATES,
		ScheduledEvent: &models.CalendarEvent{
			StartDate: primitive.NewDateTimeFromTime(start),
			EndDate:   primitive.NewDateTimeFromTime(start.Add(2 * time.Hour)),
		},
		GatheringReminder:   &models.GatheringReminder{Enabled: true, LeadTimeHours: 24, SentAt: &sentAt},
		GatheringRecurrence: &models.GatheringRecurrence{Frequency: models.RecurrenceWeekly},
		Rsvps: map[string]*models.Rsvp{
			"Last Cycle Guest": {Status: models.RsvpGoing, Email: "guest@example.com"},
		},
	})
	if err != nil {
		t.Fatalf("insert event: %v", err)
	}
	defer db.EventsCollection.DeleteOne(ctx, bson.M{"_id": eventId})

	advanceRecurringGatherings(now)

	var reloaded models.Event
	if err := db.EventsCollection.FindOne(ctx, bson.M{"_id": eventId}).Decode(&reloaded); err != nil {
		t.Fatalf("reload: %v", err)
	}

	newStart := reloaded.ScheduledEvent.StartDate.Time()
	if !newStart.After(now) {
		t.Errorf("expected next occurrence in the future, got %s (now %s)", newStart, now)
	}
	// The rolled-forward occurrence keeps the original time-of-day + 2h duration.
	if got := reloaded.ScheduledEvent.EndDate.Time().Sub(newStart); got != 2*time.Hour {
		t.Errorf("expected 2h duration preserved, got %s", got)
	}
	// Weekly cadence: the next start is start + 7*k, landing 6 days out from now.
	// Compare at primitive.DateTime (ms) precision — Mongo truncates nanoseconds.
	if want := primitive.NewDateTimeFromTime(start.AddDate(0, 0, 14)); reloaded.ScheduledEvent.StartDate != want {
		t.Errorf("next occurrence = %s, want %s", newStart, want.Time())
	}
	// Prior cycle's RSVPs cleared for a fresh headcount.
	if len(reloaded.Rsvps) != 0 {
		t.Errorf("expected RSVPs cleared on advance, got %v", reloaded.Rsvps)
	}
	// Reminder re-armed.
	if reloaded.GatheringReminder == nil || reloaded.GatheringReminder.SentAt != nil {
		t.Errorf("expected reminder re-armed (sentAt nil), got %+v", reloaded.GatheringReminder)
	}

	// Idempotent: a second tick sees the future occurrence and does nothing.
	advanceRecurringGatherings(now)
	var again models.Event
	db.EventsCollection.FindOne(ctx, bson.M{"_id": eventId}).Decode(&again)
	if !again.ScheduledEvent.StartDate.Time().Equal(newStart) {
		t.Errorf("second advance moved a future occurrence: %s -> %s", newStart, again.ScheduledEvent.StartDate.Time())
	}
}

// A series past its Until date is not advanced (stays on its last occurrence).
func TestAdvanceRecurringGatherings_RespectsUntil(t *testing.T) {
	requireDB(t)
	ctx := context.Background()

	eventId := primitive.NewObjectID()
	now := time.Now()
	start := now.AddDate(0, 0, -8)
	startDT := primitive.NewDateTimeFromTime(start)
	until := primitive.NewDateTimeFromTime(now.AddDate(0, 0, -2)) // series ended 2 days ago

	_, err := db.EventsCollection.InsertOne(ctx, models.Event{
		Id:   eventId,
		Name: "Ended Series",
		Type: models.SPECIFIC_DATES,
		ScheduledEvent: &models.CalendarEvent{
			StartDate: startDT,
			EndDate:   primitive.NewDateTimeFromTime(start.Add(2 * time.Hour)),
		},
		GatheringRecurrence: &models.GatheringRecurrence{Frequency: models.RecurrenceWeekly, Until: &until},
	})
	if err != nil {
		t.Fatalf("insert event: %v", err)
	}
	defer db.EventsCollection.DeleteOne(ctx, bson.M{"_id": eventId})

	advanceRecurringGatherings(now)

	var reloaded models.Event
	db.EventsCollection.FindOne(ctx, bson.M{"_id": eventId}).Decode(&reloaded)
	if reloaded.ScheduledEvent.StartDate != startDT {
		t.Errorf("exhausted series should not advance: %s -> %s", startDT.Time(), reloaded.ScheduledEvent.StartDate.Time())
	}
}

// A gathering whose current occurrence hasn't ended yet is left alone.
func TestAdvanceRecurringGatherings_SkipsOngoing(t *testing.T) {
	requireDB(t)
	ctx := context.Background()

	eventId := primitive.NewObjectID()
	now := time.Now()
	start := now.Add(1 * time.Hour) // future occurrence
	startDT := primitive.NewDateTimeFromTime(start)

	_, err := db.EventsCollection.InsertOne(ctx, models.Event{
		Id:   eventId,
		Name: "Upcoming Weekly",
		Type: models.SPECIFIC_DATES,
		ScheduledEvent: &models.CalendarEvent{
			StartDate: startDT,
			EndDate:   primitive.NewDateTimeFromTime(start.Add(2 * time.Hour)),
		},
		GatheringRecurrence: &models.GatheringRecurrence{Frequency: models.RecurrenceWeekly},
	})
	if err != nil {
		t.Fatalf("insert event: %v", err)
	}
	defer db.EventsCollection.DeleteOne(ctx, bson.M{"_id": eventId})

	advanceRecurringGatherings(now)

	var reloaded models.Event
	db.EventsCollection.FindOne(ctx, bson.M{"_id": eventId}).Decode(&reloaded)
	if reloaded.ScheduledEvent.StartDate != startDT {
		t.Errorf("future occurrence should not advance: %s -> %s", startDT.Time(), reloaded.ScheduledEvent.StartDate.Time())
	}
}
