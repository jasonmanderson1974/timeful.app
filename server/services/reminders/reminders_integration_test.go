package reminders

import (
	"context"
	"os"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"schej.it/server/db"
	"schej.it/server/logger"
	"schej.it/server/models"
	"schej.it/server/utils"
)

// dbReady is true when MONGODB_URI is set so the whole reminder pipeline can be
// exercised against a real Mongo. Without it these tests skip (the pure-logic
// tests in reminders_test.go still run). CI sets MONGODB_URI + a Mongo service.
var dbReady bool

func TestMain(m *testing.M) {
	logger.Init(os.Stderr)
	if os.Getenv("MONGODB_URI") != "" {
		closeConn := db.Init()
		dbReady = true
		code := m.Run()
		closeConn()
		os.Exit(code)
	}
	os.Exit(m.Run())
}

func requireDB(t *testing.T) {
	t.Helper()
	if !dbReady {
		t.Skip("MONGODB_URI not set; skipping DB-backed reminder test")
	}
}

// Exercises the full pipeline: GetEventsWithPendingReminders -> due filter ->
// collectRecipientEmails -> send -> MarkGatheringReminderSent, against real Mongo.
func TestProcessDueReminders_SendsAndMarksSent(t *testing.T) {
	requireDB(t)
	ctx := context.Background()

	eventId := primitive.NewObjectID()
	now := time.Now()
	// Gathering is 1h out; lead time 2h -> reminder is due now.
	start := now.Add(1 * time.Hour)
	desc := "Bring cigars."

	_, err := db.EventsCollection.InsertOne(ctx, models.Event{
		Id:          eventId,
		Name:        "Integration Gathering",
		Description: &desc,
		Type:        models.SPECIFIC_DATES,
		ScheduledEvent: &models.CalendarEvent{
			Summary:   "Integration Gathering",
			StartDate: primitive.NewDateTimeFromTime(start),
			EndDate:   primitive.NewDateTimeFromTime(start.Add(2 * time.Hour)),
		},
		GatheringReminder: &models.GatheringReminder{
			Enabled:       true,
			LeadTimeHours: 2,
			Timezone:      "America/Los_Angeles",
			// SentAt nil -> pending
		},
	})
	if err != nil {
		t.Fatalf("insert event: %v", err)
	}
	defer func() {
		db.EventsCollection.DeleteOne(ctx, bson.M{"_id": eventId})
		db.EventResponsesCollection.DeleteMany(ctx, bson.M{"eventId": eventId})
	}()

	// Two guest responses: one with an email (should be reminded), one without.
	_, _ = db.EventResponsesCollection.InsertMany(ctx, []interface{}{
		models.EventResponse{Id: primitive.NewObjectID(), EventId: eventId, UserId: "Guest Greg", Response: &models.Response{Name: "Guest Greg", Email: "greg@example.com"}},
		models.EventResponse{Id: primitive.NewObjectID(), EventId: eventId, UserId: "Guest Nomail", Response: &models.Response{Name: "Guest Nomail"}},
	})

	// Capture sends instead of hitting SMTP.
	var sent []string
	mockSend := func(to, subject, body, contentType string) error {
		sent = append(sent, to)
		if contentType != "text/html" {
			t.Errorf("expected text/html, got %q", contentType)
		}
		return nil
	}

	processDueReminders(now, mockSend)

	if len(sent) != 1 || sent[0] != "greg@example.com" {
		t.Fatalf("expected exactly [greg@example.com] reminded, got %v", sent)
	}

	// The event must now be marked sent (so a second tick is a no-op).
	var reloaded models.Event
	if err := db.EventsCollection.FindOne(ctx, bson.M{"_id": eventId}).Decode(&reloaded); err != nil {
		t.Fatalf("reload event: %v", err)
	}
	if reloaded.GatheringReminder == nil || reloaded.GatheringReminder.SentAt == nil {
		t.Fatalf("gatheringReminder.sentAt was not set")
	}

	// Second run must not re-send.
	sent = nil
	processDueReminders(now, mockSend)
	if len(sent) != 0 {
		t.Fatalf("second run should not re-send, got %v", sent)
	}
}

// A gathering whose reminder window hasn't opened yet must not be sent.
func TestProcessDueReminders_NotYetDue(t *testing.T) {
	requireDB(t)
	ctx := context.Background()

	eventId := primitive.NewObjectID()
	now := time.Now()
	start := now.Add(48 * time.Hour) // far out; 2h lead -> not due

	_, err := db.EventsCollection.InsertOne(ctx, models.Event{
		Id:   eventId,
		Name: "Future Gathering",
		Type: models.SPECIFIC_DATES,
		ScheduledEvent: &models.CalendarEvent{
			StartDate: primitive.NewDateTimeFromTime(start),
			EndDate:   primitive.NewDateTimeFromTime(start.Add(2 * time.Hour)),
		},
		GatheringReminder: &models.GatheringReminder{Enabled: true, LeadTimeHours: 2},
	})
	if err != nil {
		t.Fatalf("insert event: %v", err)
	}
	defer db.EventsCollection.DeleteOne(ctx, bson.M{"_id": eventId})

	var sent []string
	processDueReminders(now, func(to, subject, body, contentType string) error {
		sent = append(sent, to)
		return nil
	})
	if len(sent) != 0 {
		t.Fatalf("not-yet-due gathering should not send, got %v", sent)
	}

	var reloaded models.Event
	db.EventsCollection.FindOne(ctx, bson.M{"_id": eventId}).Decode(&reloaded)
	if reloaded.GatheringReminder != nil && reloaded.GatheringReminder.SentAt != nil {
		t.Fatalf("not-yet-due gathering should not be marked sent")
	}
}

// Guard against an accidental import prune: the production sender must satisfy SendFunc.
var _ SendFunc = utils.SendEmail
