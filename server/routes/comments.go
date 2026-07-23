// Discussion-thread comments on an event (C7). Routes are registered under the
// /events group by InitEvents.
package routes

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"sirtom/server/db"
	"sirtom/server/errs"
	"sirtom/server/models"
	"sirtom/server/responses"
)

const maxCommentLength = 2000

// sanitizeCommentText trims a comment and reports whether it's usable (non-empty
// after trimming). Over-long text is truncated to maxCommentLength.
func sanitizeCommentText(text string) (string, bool) {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return "", false
	}
	if len(trimmed) > maxCommentLength {
		trimmed = trimmed[:maxCommentLength]
	}
	return trimmed, true
}

// @Summary Posts a comment to an event's discussion thread
// @Tags events
// @Accept json
// @Produce json
// @Param eventId path string true "Event ID"
// @Param payload body object{text=string,guest=bool,name=string} true "Comment text + author identity"
// @Success 200 {object} models.Comment
// @Router /events/{eventId}/comments [post]
func addComment(c *gin.Context) {
	payload := struct {
		Text  string `json:"text" binding:"required"`
		Guest *bool  `json:"guest" binding:"required"`
		Name  string `json:"name"`
	}{}
	if err := c.Bind(&payload); err != nil {
		c.JSON(http.StatusBadRequest, responses.Error{Error: err.Error()})
		return
	}

	text, ok := sanitizeCommentText(payload.Text)
	if !ok {
		c.JSON(http.StatusBadRequest, responses.Error{Error: "empty-comment"})
		return
	}

	event, eventErr := db.GetEventByEitherId(c.Param("eventId"))
	if eventErr != nil {
		c.JSON(http.StatusInternalServerError, responses.Error{Error: errs.Internal})
		return
	}
	if event == nil {
		c.JSON(http.StatusNotFound, responses.Error{Error: errs.EventNotFound})
		return
	}

	key, _, keyOk := responderKey(c, *payload.Guest, payload.Name)
	if !keyOk {
		return
	}

	// Resolve the display name.
	authorName := payload.Name
	if !*payload.Guest {
		if user, err := db.GetUserById(key); err == nil && user != nil {
			authorName = fmt.Sprintf("%s %s", user.FirstName, user.LastName)
		}
	}

	comment := models.Comment{
		Id:         primitive.NewObjectID(),
		EventId:    event.Id,
		UserId:     key,
		IsGuest:    *payload.Guest,
		AuthorName: strings.TrimSpace(authorName),
		Text:       text,
		CreatedAt:  primitive.NewDateTimeFromTime(time.Now()),
	}
	if err := db.InsertComment(comment); err != nil {
		c.JSON(http.StatusInternalServerError, responses.Error{Error: errs.Internal})
		return
	}

	c.JSON(http.StatusOK, comment)
}

// @Summary Edits the caller's own comment
// @Tags events
// @Accept json
// @Produce json
// @Param eventId path string true "Event ID"
// @Param commentId path string true "Comment ID"
// @Param payload body object{text=string,guest=bool,name=string} true "New text + author identity"
// @Success 200
// @Router /events/{eventId}/comments/{commentId} [put]
func editComment(c *gin.Context) {
	payload := struct {
		Text  string `json:"text" binding:"required"`
		Guest *bool  `json:"guest" binding:"required"`
		Name  string `json:"name"`
	}{}
	if err := c.Bind(&payload); err != nil {
		c.JSON(http.StatusBadRequest, responses.Error{Error: err.Error()})
		return
	}

	text, ok := sanitizeCommentText(payload.Text)
	if !ok {
		c.JSON(http.StatusBadRequest, responses.Error{Error: "empty-comment"})
		return
	}

	comment, err := db.GetCommentById(c.Param("commentId"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, responses.Error{Error: errs.Internal})
		return
	}
	if comment == nil {
		c.JSON(http.StatusNotFound, responses.Error{Error: "comment-not-found"})
		return
	}

	key, _, keyOk := responderKey(c, *payload.Guest, payload.Name)
	if !keyOk {
		return
	}

	// Editing is own-only.
	if comment.UserId != key || comment.IsGuest != *payload.Guest {
		c.JSON(http.StatusForbidden, responses.Error{Error: errs.NotAuthorized})
		return
	}

	if err := db.UpdateCommentText(comment.Id, text, primitive.NewDateTimeFromTime(time.Now())); err != nil {
		c.JSON(http.StatusInternalServerError, responses.Error{Error: errs.Internal})
		return
	}

	c.Status(http.StatusOK)
}

// @Summary Deletes a comment (own, or any when the caller is the event owner)
// @Tags events
// @Accept json
// @Produce json
// @Param eventId path string true "Event ID"
// @Param commentId path string true "Comment ID"
// @Param payload body object{guest=bool,name=string} true "Author identity"
// @Success 200
// @Router /events/{eventId}/comments/{commentId} [delete]
func deleteComment(c *gin.Context) {
	payload := struct {
		Guest *bool  `json:"guest" binding:"required"`
		Name  string `json:"name"`
	}{}
	if err := c.Bind(&payload); err != nil {
		c.JSON(http.StatusBadRequest, responses.Error{Error: err.Error()})
		return
	}

	comment, err := db.GetCommentById(c.Param("commentId"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, responses.Error{Error: errs.Internal})
		return
	}
	if comment == nil {
		c.Status(http.StatusOK) // already gone — idempotent
		return
	}

	key, _, keyOk := responderKey(c, *payload.Guest, payload.Name)
	if !keyOk {
		return
	}

	// Allowed if it's the caller's own comment, or the caller is the event owner.
	isOwn := comment.UserId == key && comment.IsGuest == *payload.Guest
	isOwner := false
	if event, eventErr := db.GetEventByEitherId(c.Param("eventId")); eventErr == nil && event != nil {
		session := sessions.Default(c)
		if uid, signedIn := session.Get("userId").(string); signedIn {
			isOwner = event.OwnerId != primitive.NilObjectID && event.OwnerId.Hex() == uid
		}
	}
	if !isOwn && !isOwner {
		c.JSON(http.StatusForbidden, responses.Error{Error: errs.NotAuthorized})
		return
	}

	if err := db.DeleteComment(comment.Id); err != nil {
		c.JSON(http.StatusInternalServerError, responses.Error{Error: errs.Internal})
		return
	}

	c.Status(http.StatusOK)
}
