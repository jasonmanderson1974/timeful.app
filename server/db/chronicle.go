package db

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"sirtom/server/logger"
	"sirtom/server/models"
)

// InsertChronicleEntry records a completed-gathering snapshot (C10). A duplicate
// (same eventId + startDate — the unique index) is treated as a no-op success,
// so racing scheduler ticks / re-runs can't create duplicate history.
func InsertChronicleEntry(entry models.ChronicleEntry) error {
	_, err := ChronicleCollection.InsertOne(context.Background(), entry)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return nil // already captured — fine
		}
		logger.StdErr.Println(err)
		return err
	}
	return nil
}

// GetChronicleEntries returns captured gatherings, most recent first, capped at
// limit (<=0 means a sane default).
func GetChronicleEntries(limit int) ([]models.ChronicleEntry, error) {
	if limit <= 0 {
		limit = 200
	}
	result, err := ChronicleCollection.Find(
		context.Background(),
		bson.M{},
		options.Find().SetSort(bson.M{"startDate": -1}).SetLimit(int64(limit)),
	)
	if err != nil {
		logger.StdErr.Println(err)
		return []models.ChronicleEntry{}, err
	}

	entries := make([]models.ChronicleEntry, 0)
	if err := result.All(context.Background(), &entries); err != nil {
		logger.StdErr.Println(err)
		return []models.ChronicleEntry{}, err
	}
	return entries, nil
}

// GetPastNonRecurringGatheringsToArchive returns one-off (non-recurring)
// gatherings whose confirmed time has passed and that haven't been captured into
// the Chronicle yet. Recurring gatherings are captured per-occurrence at advance
// time instead (see services/reminders.advanceRecurringGatherings), so they're
// excluded here. (C10)
func GetPastNonRecurringGatheringsToArchive(now primitive.DateTime) ([]models.Event, error) {
	result, err := EventsCollection.Find(context.Background(), bson.M{
		"gatheringRecurrence.frequency": bson.M{"$exists": false},
		"scheduledEvent.endDate":        bson.M{"$lte": now},
		"chronicled":                    bson.M{"$ne": true},
		"isDeleted":                     bson.M{"$ne": true},
	})
	if err != nil {
		logger.StdErr.Println(err)
		return []models.Event{}, err
	}

	var events []models.Event
	if err := result.All(context.Background(), &events); err != nil {
		logger.StdErr.Println(err)
		return []models.Event{}, err
	}
	return events, nil
}

// MarkEventChronicled flags a (non-recurring) event as captured so it isn't
// re-snapshotted on the next tick. Idempotent. (C10)
func MarkEventChronicled(eventId primitive.ObjectID) error {
	_, err := EventsCollection.UpdateByID(context.Background(), eventId,
		bson.M{"$set": bson.M{"chronicled": true}})
	if err != nil {
		logger.StdErr.Println(err)
	}
	return err
}
