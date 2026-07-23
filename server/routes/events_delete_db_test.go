package routes

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"sirtom/server/db"
	"sirtom/server/models"
)

// TestDeleteEvent_ByShortId verifies deleteEvent resolves the event by its
// short id, not only the Mongo _id (the E2 fix). Before, a short id 400'd
// while every other event route accepted either id via GetEventByEitherId.
func TestDeleteEvent_ByShortId(t *testing.T) {
	requireDB(t)
	ctx := context.Background()

	owner := &models.User{Id: primitive.NewObjectID()}
	eventId := primitive.NewObjectID()
	shortId := "e2tst"
	if _, err := db.EventsCollection.InsertOne(ctx, models.Event{
		Id:      eventId,
		ShortId: &shortId,
		Type:    models.DOW,
		OwnerId: owner.Id,
	}); err != nil {
		t.Fatalf("insert event: %v", err)
	}
	defer cleanupEvent(eventId)

	r := newTestRouter()
	r.Use(func(c *gin.Context) { c.Set("authUser", owner); c.Next() })
	r.DELETE("/events/:eventId", deleteEvent)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/events/"+shortId, nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("delete by short id: got %d, want 200 (body: %s)", w.Code, w.Body.String())
	}

	count, err := db.EventsCollection.CountDocuments(ctx, bson.M{"_id": eventId})
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 0 {
		t.Fatalf("event not deleted by short id: still %d docs", count)
	}
}

// TestDeleteEvent_NotFound verifies a well-formed-but-unknown id now 404s
// (previously an unknown _id fell through to a decode error / 500, and a short
// id 400'd before resolution).
func TestDeleteEvent_NotFound(t *testing.T) {
	requireDB(t)

	owner := &models.User{Id: primitive.NewObjectID()}
	r := newTestRouter()
	r.Use(func(c *gin.Context) { c.Set("authUser", owner); c.Next() })
	r.DELETE("/events/:eventId", deleteEvent)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/events/nope404", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("unknown id: got %d, want 404 (body: %s)", w.Code, w.Body.String())
	}
}
