package routes

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"sirtom/server/db"
	"sirtom/server/models"
)

// dbReady is set by TestMain (admin_handlers_test.go) when MONGODB_URI is
// configured. DB-backed tests skip when it isn't, so `go test ./routes/` still
// passes on a machine without Mongo — only the DB-free tests run there. CI sets
// MONGODB_URI and runs a Mongo service, so these run in CI.
var dbReady bool

func requireDB(t *testing.T) {
	t.Helper()
	if !dbReady {
		t.Skip("MONGODB_URI not set; skipping DB-backed handler test")
	}
}

func intPtrTest(i int) *int { return &i }

func float32PtrTest(f float32) *float32 { return &f }

// newTestRouter builds a gin engine with the session middleware the response
// handlers rely on (sessions.Default panics without it).
func newTestRouter() *gin.Engine {
	store := cookie.NewStore([]byte("test-session-secret-at-least-32-bytes!"))
	r := gin.New()
	r.Use(sessions.Sessions("session", store))
	return r
}

func cleanupEvent(eventId primitive.ObjectID) {
	ctx := context.Background()
	db.EventsCollection.DeleteOne(ctx, bson.M{"_id": eventId})
	db.EventResponsesCollection.DeleteMany(ctx, bson.M{"eventId": eventId})
}

func TestGetResponses_EventNotFound(t *testing.T) {
	requireDB(t)
	r := newTestRouter()
	r.GET("/events/:eventId/responses", getResponses)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet,
		"/events/000000000000000000000000/responses?timeMin=2025-01-01T00:00:00Z&timeMax=2025-01-02T00:00:00Z", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("event not found: got %d, want 404 (body: %s)", w.Code, w.Body.String())
	}
}

func TestGetResponses_ReturnsAllWhenBlindOff(t *testing.T) {
	requireDB(t)
	ctx := context.Background()

	eventId := primitive.NewObjectID()
	if _, err := db.EventsCollection.InsertOne(ctx, models.Event{
		Id:           eventId,
		Type:         models.DOW,
		OwnerId:      primitive.NewObjectID(),
		NumResponses: intPtrTest(2),
	}); err != nil {
		t.Fatalf("insert event: %v", err)
	}
	defer cleanupEvent(eventId)

	slot := primitive.NewDateTimeFromTime(time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC))
	for _, name := range []string{"alice", "bob"} {
		if _, err := db.EventResponsesCollection.InsertOne(ctx, models.EventResponse{
			Id:      primitive.NewObjectID(),
			EventId: eventId,
			UserId:  name,
			Response: &models.Response{
				Name:         name,
				Email:        name + "@example.test",
				Availability: []primitive.DateTime{slot},
			},
		}); err != nil {
			t.Fatalf("insert response %s: %v", name, err)
		}
	}

	r := newTestRouter()
	r.GET("/events/:eventId/responses", getResponses)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet,
		"/events/"+eventId.Hex()+"/responses?timeMin=2025-01-01T00:00:00Z&timeMax=2025-01-02T00:00:00Z", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("got %d, want 200 (body: %s)", w.Code, w.Body.String())
	}
	var got map[string]json.RawMessage
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal body: %v (%s)", err, w.Body.String())
	}
	if len(got) != 2 {
		t.Fatalf("got %d responses, want 2 (blind off returns all)", len(got))
	}
}

func TestUpdateEventResponse_GuestCreatesResponse(t *testing.T) {
	requireDB(t)
	ctx := context.Background()

	eventId := primitive.NewObjectID()
	if _, err := db.EventsCollection.InsertOne(ctx, models.Event{
		Id:           eventId,
		Type:         models.DOW,
		OwnerId:      primitive.NewObjectID(),
		Duration:     float32PtrTest(1),
		NumResponses: intPtrTest(0),
	}); err != nil {
		t.Fatalf("insert event: %v", err)
	}
	defer cleanupEvent(eventId)

	r := newTestRouter()
	r.POST("/events/:eventId/response", updateEventResponse)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/events/"+eventId.Hex()+"/response",
		strings.NewReader(`{"guest":true,"name":"Zoe"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("got %d, want 200 (body: %s)", w.Code, w.Body.String())
	}

	count, err := db.EventResponsesCollection.CountDocuments(ctx, bson.M{"eventId": eventId, "userId": "Zoe"})
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 1 {
		t.Fatalf("guest response not persisted: got %d docs, want 1", count)
	}
}
