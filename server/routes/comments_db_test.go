package routes

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"sirtom/server/db"
	"sirtom/server/models"
)

// commentsTestRouter wires the comment routes (+ a test-login) onto a gin engine
// with the session middleware the handlers rely on.
func commentsTestRouter() *gin.Engine {
	r := newTestRouter()
	registerTestLogin(r)
	r.POST("/events/:eventId/comments", addComment)
	r.PUT("/events/:eventId/comments/:commentId", editComment)
	r.DELETE("/events/:eventId/comments/:commentId", deleteComment)
	return r
}

func TestComments_GuestPostEditDelete(t *testing.T) {
	requireDB(t)
	ctx := context.Background()

	eventId := primitive.NewObjectID()
	if _, err := db.EventsCollection.InsertOne(ctx, models.Event{Id: eventId, Type: models.SPECIFIC_DATES}); err != nil {
		t.Fatalf("insert event: %v", err)
	}
	defer func() {
		db.EventsCollection.DeleteOne(ctx, bson.M{"_id": eventId})
		db.CommentsCollection.DeleteMany(ctx, bson.M{"eventId": eventId})
	}()

	h := commentsTestRouter()

	// Empty text -> 400.
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/events/"+eventId.Hex()+"/comments",
		strings.NewReader(`{"guest":true,"name":"Greg","text":"   "}`))
	req.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("empty text: got %d, want 400", w.Code)
	}

	// Guest posts a real comment -> 200, returns the comment.
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/events/"+eventId.Hex()+"/comments",
		strings.NewReader(`{"guest":true,"name":"Greg","text":"Parking is out back"}`))
	req.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("guest post: got %d, want 200 (body: %s)", w.Code, w.Body.String())
	}
	var created models.Comment
	json.Unmarshal(w.Body.Bytes(), &created)
	if created.AuthorName != "Greg" || created.Text != "Parking is out back" || !created.IsGuest {
		t.Fatalf("unexpected created comment: %+v", created)
	}
	cid := created.Id.Hex()

	// Owner (event has no owner here; the guest is the author) edits own -> updatedAt set.
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPut, "/events/"+eventId.Hex()+"/comments/"+cid,
		strings.NewReader(`{"guest":true,"name":"Greg","text":"Parking is out back, gate code 1234"}`))
	req.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("edit own: got %d, want 200 (body: %s)", w.Code, w.Body.String())
	}
	edited, _ := db.GetCommentById(cid)
	if edited == nil || edited.UpdatedAt == nil || !strings.Contains(edited.Text, "gate code") {
		t.Fatalf("edit did not persist / set updatedAt: %+v", edited)
	}

	// A different guest cannot delete Greg's comment -> 403.
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodDelete, "/events/"+eventId.Hex()+"/comments/"+cid,
		strings.NewReader(`{"guest":true,"name":"Mallory"}`))
	req.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("other guest delete: got %d, want 403", w.Code)
	}

	// Greg deletes his own -> 200, gone.
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodDelete, "/events/"+eventId.Hex()+"/comments/"+cid,
		strings.NewReader(`{"guest":true,"name":"Greg"}`))
	req.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("delete own: got %d, want 200", w.Code)
	}
	if gone, _ := db.GetCommentById(cid); gone != nil {
		t.Fatal("comment should be deleted")
	}
}

// The event owner can delete another author's comment (moderation).
func TestComments_OwnerDeletesAny(t *testing.T) {
	requireDB(t)
	ctx := context.Background()

	ownerId := primitive.NewObjectID()
	if _, err := db.UsersCollection.InsertOne(ctx, models.User{Id: ownerId, Email: "owner@example.test"}); err != nil {
		t.Fatalf("insert owner: %v", err)
	}
	defer deleteTestUser(ownerId)

	eventId := primitive.NewObjectID()
	if _, err := db.EventsCollection.InsertOne(ctx, models.Event{Id: eventId, OwnerId: ownerId, Type: models.SPECIFIC_DATES}); err != nil {
		t.Fatalf("insert event: %v", err)
	}
	defer func() {
		db.EventsCollection.DeleteOne(ctx, bson.M{"_id": eventId})
		db.CommentsCollection.DeleteMany(ctx, bson.M{"eventId": eventId})
	}()

	// A guest comment by someone else.
	commentId := primitive.NewObjectID()
	db.CommentsCollection.InsertOne(ctx, models.Comment{
		Id: commentId, EventId: eventId, UserId: "Greg", IsGuest: true,
		AuthorName: "Greg", Text: "hello",
	})

	h := commentsTestRouter()
	cookie := loginAs(t, h, ownerId.Hex())

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/events/"+eventId.Hex()+"/comments/"+commentId.Hex(),
		strings.NewReader(`{"guest":false}`))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(cookie)
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("owner delete of another's comment: got %d, want 200 (body: %s)", w.Code, w.Body.String())
	}
	if gone, _ := db.GetCommentById(commentId.Hex()); gone != nil {
		t.Fatal("owner should have deleted the comment")
	}
}
